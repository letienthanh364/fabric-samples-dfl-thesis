package transport

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nebula/gateway/internal/common"
	"github.com/nebula/gateway/internal/statecontract/model"
)

// Transport issues Fabric CLI requests for the state contract.
type Transport struct {
	fabric *common.FabricClient
}

// NewTransport returns a Transport.
func NewTransport(fabric *common.FabricClient) *Transport {
	return &Transport{fabric: fabric}
}

func (t *Transport) ListAssets(_ context.Context, peer string) ([]model.Asset, error) {
	raw, err := t.fabric.QueryChaincode(peer, []string{"GetAllAssets"})
	if err != nil {
		return nil, err
	}
	var assets []model.Asset
	if err := json.Unmarshal(raw, &assets); err != nil {
		return nil, fmt.Errorf("unable to decode ledger response: %w", err)
	}
	return assets, nil
}

func (t *Transport) CreateAsset(_ context.Context, peer string, asset model.Asset) error {
	args := []string{
		"CreateAsset",
		asset.ID,
		asset.Color,
		fmt.Sprintf("%d", asset.Size),
		asset.Owner,
		fmt.Sprintf("%d", asset.AppraisedValue),
	}
	return t.fabric.InvokeChaincode(peer, args)
}
