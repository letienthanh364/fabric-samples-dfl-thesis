package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nebula/gateway/internal/common"
	"github.com/nebula/gateway/internal/jobcontract/model"
)

// Transport issues Fabric CLI commands for the job contract.
type Transport struct {
	fabric *common.FabricClient
}

// NewTransport wires a Transport with the fabric client.
func NewTransport(fabric *common.FabricClient) *Transport {
	return &Transport{fabric: fabric}
}

func (t *Transport) UpsertGenesisModelCID(_ context.Context, peer string, payload model.GenesisModelCIDRequest) error {
	args := []string{
		"UpsertGenesisModelCID",
		payload.JobID,
		payload.CID,
		payload.Purpose,
		payload.ModelFamily,
		payload.DatasetSummary,
		payload.Notes,
	}
	return t.fabric.InvokeChaincode(peer, args)
}

func (t *Transport) GetGenesisModelCID(_ context.Context, peer, jobID string) (*model.GenesisModelCIDRecord, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	raw, err := t.fabric.QueryChaincode(peer, []string{"GetGenesisModelCID", jobID})
	if err != nil {
		return nil, err
	}
	var record model.GenesisModelCIDRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("unable to decode ledger response: %w", err)
	}
	return &record, nil
}

func (t *Transport) UpsertGenesisModelHash(_ context.Context, peer string, payload model.GenesisModelHashRequest) error {
	args := []string{
		"UpsertGenesisModelHash",
		payload.JobID,
		payload.Hash,
		payload.HashAlgorithm,
		payload.ModelFormat,
		payload.Compression,
		payload.Notes,
	}
	return t.fabric.InvokeChaincode(peer, args)
}

func (t *Transport) GetGenesisModelHash(_ context.Context, peer, jobID string) (*model.GenesisModelHashRecord, error) {
	if jobID == "" {
		return nil, fmt.Errorf("jobId is required")
	}
	raw, err := t.fabric.QueryChaincode(peer, []string{"GetGenesisModelHash", jobID})
	if err != nil {
		return nil, err
	}
	var record model.GenesisModelHashRecord
	if err := json.Unmarshal(raw, &record); err != nil {
		return nil, fmt.Errorf("unable to decode ledger response: %w", err)
	}
	return &record, nil
}
