package controller

import (
	"net/http"

	"github.com/nebula/gateway/internal/common"
	"github.com/nebula/gateway/internal/didcontract/service"
)

// Handler exposes placeholder DID contract endpoints.
type Handler struct {
	svc *service.Service
}

// NewHandler returns a Handler.
func NewHandler(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the placeholder route.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/did-contract", h.handlePlaceholder)
}

func (h *Handler) handlePlaceholder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	common.WriteJSON(w, http.StatusNotImplemented, map[string]string{
		"message": h.svc.PlaceholderMessage(),
	})
}
