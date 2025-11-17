package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nebula/gateway/internal/common"
	"github.com/nebula/gateway/internal/statecontract/model"
	"github.com/nebula/gateway/internal/statecontract/service"
)

// Handler wires /assets operations.
type Handler struct {
	cfg *common.Config
	svc *service.Service
}

// NewHandler creates a Handler.
func NewHandler(cfg *common.Config, svc *service.Service) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

// RegisterRoutes mounts the /assets endpoint.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/assets", h.handleAssets)
}

func (h *Handler) handleAssets(w http.ResponseWriter, r *http.Request) {
	peer := h.cfg.ResolvePeer(r.URL.Query().Get("peer"))
	switch r.Method {
	case http.MethodGet:
		assets, err := h.svc.ListAssets(r.Context(), peer)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, assets)
	case http.MethodPost:
		var asset model.Asset
		if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}
		if err := h.svc.CreateAsset(r.Context(), peer, asset); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]string{"id": asset.ID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
