package service

import (
	"context"

	"github.com/nebula/gateway/internal/jobcontract/model"
	"github.com/nebula/gateway/internal/jobcontract/transport"
)

// Service coordinates job contract operations.
type Service struct {
	transport *transport.Transport
}

// NewService returns a Service instance.
func NewService(t *transport.Transport) *Service {
	return &Service{transport: t}
}

func (s *Service) UpsertGenesisModelCID(ctx context.Context, peer string, payload model.GenesisModelCIDRequest) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	return s.transport.UpsertGenesisModelCID(ctx, peer, payload)
}

func (s *Service) GetGenesisModelCID(ctx context.Context, peer, jobID string) (*model.GenesisModelCIDRecord, error) {
	return s.transport.GetGenesisModelCID(ctx, peer, jobID)
}

func (s *Service) UpsertGenesisModelHash(ctx context.Context, peer string, payload model.GenesisModelHashRequest) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	return s.transport.UpsertGenesisModelHash(ctx, peer, payload)
}

func (s *Service) GetGenesisModelHash(ctx context.Context, peer, jobID string) (*model.GenesisModelHashRecord, error) {
	return s.transport.GetGenesisModelHash(ctx, peer, jobID)
}
