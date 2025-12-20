package common

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"sync/atomic"
	"time"
)

// FabricClient shells out to the Fabric peer CLI to submit/evaluate chaincode transactions.
type FabricClient struct {
	cfg       *Config
	peerNames []string
	peerIndex uint32
}

// NewFabricClient wires a FabricClient with the gateway configuration.
func NewFabricClient(cfg *Config) *FabricClient {
	return &FabricClient{cfg: cfg, peerNames: buildPeerOrder(cfg)}
}

// Config exposes the underlying configuration.
func (f *FabricClient) Config() *Config {
	return f.cfg
}

// WaitForChannelReady ensures at least one peer has joined the channel before serving traffic.
func (f *FabricClient) WaitForChannelReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	peerNames := f.peerNames
	if len(peerNames) == 0 {
		return fmt.Errorf("no peers configured")
	}

	var lastErr error
	for time.Now().Before(deadline) {
		for _, peerName := range peerNames {
			if _, err := f.runPeerCommand(peerName, "", []string{"channel", "getinfo", "-c", f.cfg.Channel}); err == nil {
				return nil
			} else {
				lastErr = err
			}
		}
		time.Sleep(5 * time.Second)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("channel readiness timed out")
	}
	return lastErr
}

// QueryChaincode evaluates the provided function/args on the target peer.
func (f *FabricClient) QueryChaincode(peerName, identity string, args []string) ([]byte, error) {
	payload := map[string]any{"Args": args}
	return f.runPeerCommand(peerName, identity, []string{
		"chaincode", "query",
		"-C", f.cfg.Channel,
		"-n", f.cfg.Chaincode,
		"-c", MustJSON(payload),
	})
}

// InvokeChaincode submits a proposal and waits for commit.
func (f *FabricClient) InvokeChaincode(peerName, identity string, args []string) error {
	payload := map[string]any{"Args": args}
	_, err := f.runPeerCommand(peerName, identity, []string{
		"chaincode", "invoke",
		"-o", f.cfg.OrdererEndpoint,
		"--ordererTLSHostnameOverride", f.cfg.OrdererHost,
		"-C", f.cfg.Channel,
		"-n", f.cfg.Chaincode,
		"--waitForEvent",
		"--tls",
		"--cafile", f.cfg.OrdererTLSCA,
		"--peerAddresses", f.cfg.Peers[peerName].Address,
		"--tlsRootCertFiles", f.cfg.Peers[peerName].TLSPath,
		"-c", MustJSON(payload),
	})
	return err
}

// SelectPeer returns the next peer using a round-robin strategy.
func (f *FabricClient) SelectPeer() string {
	if len(f.peerNames) == 0 {
		return ""
	}
	idx := atomic.AddUint32(&f.peerIndex, 1)
	pos := int((idx - 1) % uint32(len(f.peerNames)))
	return f.peerNames[pos]
}

func (f *FabricClient) runPeerCommand(peerName, identity string, args []string) ([]byte, error) {
	peerCfg, ok := f.cfg.Peers[peerName]
	if !ok {
		return nil, fmt.Errorf("peer %s is not configured", peerName)
	}
	mspPath, err := f.cfg.MSPPathForIdentity(identity)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("peer", args...)
	env := append(os.Environ(),
		fmt.Sprintf("CORE_PEER_LOCALMSPID=%s", f.cfg.MSPID),
		fmt.Sprintf("CORE_PEER_MSPCONFIGPATH=%s", mspPath),
		"CORE_PEER_TLS_ENABLED=true",
		fmt.Sprintf("CORE_PEER_TLS_ROOTCERT_FILE=%s", peerCfg.TLSPath),
		fmt.Sprintf("CORE_PEER_ADDRESS=%s", peerCfg.Address),
		fmt.Sprintf("FABRIC_CFG_PATH=%s", f.cfg.FabricCfgPath),
	)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleaned := SanitizeCLIError(string(output))
		return nil, fmt.Errorf("peer command failed: %s", cleaned)
	}
	return bytes.TrimSpace(output), nil
}

func buildPeerOrder(cfg *Config) []string {
	if len(cfg.Peers) == 0 {
		return nil
	}
	names := make([]string, 0, len(cfg.Peers))
	if cfg.DefaultPeer != "" {
		if _, ok := cfg.Peers[cfg.DefaultPeer]; ok {
			names = append(names, cfg.DefaultPeer)
		}
	}
	var remaining []string
	for name := range cfg.Peers {
		if name == cfg.DefaultPeer {
			continue
		}
		remaining = append(remaining, name)
	}
	sort.Strings(remaining)
	names = append(names, remaining...)
	return names
}
