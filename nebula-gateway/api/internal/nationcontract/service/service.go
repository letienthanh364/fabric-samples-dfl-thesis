package service

import "github.com/nebula/gateway/internal/nationcontract/transport"

// Service currently provides placeholder messaging for nation contract endpoints.
type Service struct {
	transport *transport.Transport
}

// NewService returns a Service.
func NewService(t *transport.Transport) *Service {
	return &Service{transport: t}
}

// PlaceholderMessage communicates upcoming functionality.
func (s *Service) PlaceholderMessage() string {
	return "Nation contract endpoints will be added soon"
}
