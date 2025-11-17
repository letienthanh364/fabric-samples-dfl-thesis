package service

import "github.com/nebula/gateway/internal/didcontract/transport"

// Service currently exposes placeholder functionality for DID contract endpoints.
type Service struct {
	transport *transport.Transport
}

// NewService creates a Service.
func NewService(t *transport.Transport) *Service {
	return &Service{transport: t}
}

// PlaceholderMessage communicates that the module is not implemented yet.
func (s *Service) PlaceholderMessage() string {
	return "DID contract endpoints will be added soon"
}
