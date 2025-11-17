package common

import (
	"errors"
	"fmt"
	"os"
	"strings"
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
	defaultPeer := "peer0"
	if _, ok := peers[defaultPeer]; !ok {
		for name := range peers {
			defaultPeer = name
			break
		}
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

// ResolvePeer validates the requested peer name or falls back to the default.
func (c *Config) ResolvePeer(name string) string {
	if name == "" {
		return c.DefaultPeer
	}
	if _, ok := c.Peers[name]; !ok {
		return c.DefaultPeer
	}
	return name
}

func fallbackEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}
