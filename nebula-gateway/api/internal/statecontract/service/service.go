package service

import (
	"context"

	"github.com/nebula/gateway/internal/statecontract/model"
	"github.com/nebula/gateway/internal/statecontract/transport"
)

// Service coordinates operations for the state contract.
type Service struct {
	transport *transport.Transport
}

// NewService constructs a Service.
func NewService(t *transport.Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) ListAssets(ctx context.Context, peer string) ([]model.Asset, error) {
	return s.transport.ListAssets(ctx, peer)
}

func (s *Service) CreateAsset(ctx context.Context, peer string, asset model.Asset) error {
	if err := asset.Validate(); err != nil {
		return err
	}
	return s.transport.CreateAsset(ctx, peer, asset)
}
