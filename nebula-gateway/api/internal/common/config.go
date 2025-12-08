package common

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Config describes the Fabric network settings exposed to the gateway.
type Config struct {
	Channel         string
	Chaincode       string
	MSPID           string
	MSPPath         string
	OrdererEndpoint string
	OrdererHost     string
	OrdererTLSCA    string
	FabricCfgPath   string
	Peers           map[string]PeerConfig
	DefaultPeer     string
	StatePeerRoutes map[string][]string
	AuthSecret      string

	stateRouteIndex map[string]int
	stateRouteMu    sync.Mutex
}

// PeerConfig captures the TLS material and address for an endorsing peer.
type PeerConfig struct {
	Name    string
	Address string
	TLSPath string
}

// LoadConfig builds a Config instance from environment variables.
func LoadConfig() (*Config, error) {
	channel := fallbackEnv("FABRIC_CHANNEL", "nebulachannel")
	chaincode := fallbackEnv("FABRIC_CHAINCODE", "basic")
	mspID := fallbackEnv("MSP_ID", "Org1MSP")
	orgPath := os.Getenv("ORG_CRYPTO_PATH")
	if orgPath == "" {
		return nil, errors.New("ORG_CRYPTO_PATH must be set")
	}
	admin := fallbackEnv("ADMIN_IDENTITY", "Admin@org1.nebula.com")
	mspPath := fmt.Sprintf("%s/users/%s/msp", orgPath, admin)
	ordererEndpoint := fallbackEnv("ORDERER_ENDPOINT", "orderer.nebula.com:7050")
	ordererTLS := fallbackEnv("ORDERER_TLS_CA", "/organizations/ordererOrganizations/nebula.com/orderers/orderer.nebula.com/msp/tlscacerts/tlsca.nebula.com-cert.pem")
	peerDomain := fallbackEnv("ORG_DOMAIN", "org1.nebula.com")
	fabricCfgPath := fallbackEnv("FABRIC_CFG_PATH", "/etc/hyperledger/fabric")

	peers, err := parsePeerConfig(os.Getenv("PEER_ENDPOINTS"), orgPath, peerDomain)
	if err != nil {
		return nil, err
	}
	stateRoutes, err := parseStatePeerRoutes(os.Getenv("STATE_PEER_ROUTES"), peers)
	if err != nil {
		return nil, err
	}
	defaultPeer := "peer0"
	if _, ok := peers[defaultPeer]; !ok {
		for name := range peers {
			defaultPeer = name
			break
		}
	}
	authSecret := os.Getenv("AUTH_JWT_SECRET")
	if authSecret == "" {
		return nil, errors.New("AUTH_JWT_SECRET must be set")
	}

	host, _, found := strings.Cut(ordererEndpoint, ":")
	if !found || host == "" {
		host = ordererEndpoint
	}

	return &Config{
		Channel:         channel,
		Chaincode:       chaincode,
		MSPID:           mspID,
		MSPPath:         mspPath,
		OrdererEndpoint: ordererEndpoint,
		OrdererHost:     host,
		OrdererTLSCA:    ordererTLS,
		FabricCfgPath:   fabricCfgPath,
		Peers:           peers,
		DefaultPeer:     defaultPeer,
		StatePeerRoutes: stateRoutes,
		AuthSecret:      authSecret,
		stateRouteIndex: make(map[string]int),
	}, nil
}

func parsePeerConfig(spec, orgPath, domain string) (map[string]PeerConfig, error) {
	if spec == "" {
		return nil, errors.New("PEER_ENDPOINTS must be provided")
	}
	entries := strings.Split(spec, ",")
	peers := make(map[string]PeerConfig, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer entry %s", entry)
		}
		name := parts[0]
		addr := parts[1]
		tlsPath := fmt.Sprintf("%s/peers/%s.%s/tls/ca.crt", orgPath, name, domain)
		peers[name] = PeerConfig{Name: name, Address: addr, TLSPath: tlsPath}
	}
	if len(peers) == 0 {
		return nil, errors.New("no peers configured")
	}
	return peers, nil
}

func parseStatePeerRoutes(spec string, peers map[string]PeerConfig) (map[string][]string, error) {
	if spec == "" {
		return nil, errors.New("STATE_PEER_ROUTES must be provided")
	}
	result := make(map[string][]string)
	entries := strings.Split(spec, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		state, peerList, found := strings.Cut(entry, "=")
		if !found {
			return nil, fmt.Errorf("invalid state peer route entry %s", entry)
		}
		state = strings.TrimSpace(state)
		if state == "" {
			return nil, fmt.Errorf("state identifier is required in entry %s", entry)
		}
		rawPeers := strings.Split(peerList, "|")
		var resolved []string
		for _, name := range rawPeers {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := peers[name]; !ok {
				return nil, fmt.Errorf("state %s references unknown peer %s", state, name)
			}
			resolved = append(resolved, name)
		}
		if len(resolved) < 2 {
			return nil, fmt.Errorf("state %s must be mapped to at least 2 peers", state)
		}
		result[state] = resolved
	}
	if len(result) == 0 {
		return nil, errors.New("STATE_PEER_ROUTES does not include any routes")
	}
	return result, nil
}

// PeerForState chooses the next peer assigned to the provided state using round-robin.
func (c *Config) PeerForState(state string) (string, error) {
	if state == "" {
		return "", errors.New("state is required to select a peer")
	}
	c.stateRouteMu.Lock()
	defer c.stateRouteMu.Unlock()
	peers := c.StatePeerRoutes[state]
	if len(peers) == 0 {
		return "", fmt.Errorf("state %s is not allowed to access the fabric", state)
	}
	idx := c.stateRouteIndex[state] % len(peers)
	c.stateRouteIndex[state] = (idx + 1) % len(peers)
	return peers[idx], nil
}

func fallbackEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
