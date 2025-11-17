package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/nebula/gateway/internal/common"
	"github.com/nebula/gateway/internal/jobcontract/model"
	"github.com/nebula/gateway/internal/jobcontract/service"
)

// Handler wires HTTP handlers for job contract endpoints.
type Handler struct {
	cfg *common.Config
	svc *service.Service
}

// NewHandler builds a Handler.
func NewHandler(cfg *common.Config, svc *service.Service) *Handler {
	return &Handler{cfg: cfg, svc: svc}
}

// RegisterRoutes mounts the job contract endpoints under the supplied mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/job-contract/genesis-model-cid", h.handleGenesisModelCID)
	mux.HandleFunc("/job-contract/genesis-model-hash", h.handleGenesisModelHash)
}

func (h *Handler) handleGenesisModelCID(w http.ResponseWriter, r *http.Request) {
	peer := h.cfg.ResolvePeer(r.URL.Query().Get("peer"))
	switch r.Method {
	case http.MethodGet:
		jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
		if jobID == "" {
			common.WriteErrorWithCode(w, http.StatusBadRequest, errJobIDRequired)
			return
		}
		record, err := h.svc.GetGenesisModelCID(r.Context(), peer, jobID)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, record)
	case http.MethodPost:
		var payload model.GenesisModelCIDRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}
		if err := h.svc.UpsertGenesisModelCID(r.Context(), peer, payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]string{"jobId": payload.JobID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleGenesisModelHash(w http.ResponseWriter, r *http.Request) {
	peer := h.cfg.ResolvePeer(r.URL.Query().Get("peer"))
	switch r.Method {
	case http.MethodGet:
		jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
		if jobID == "" {
			common.WriteErrorWithCode(w, http.StatusBadRequest, errJobIDRequired)
			return
		}
		record, err := h.svc.GetGenesisModelHash(r.Context(), peer, jobID)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, record)
	case http.MethodPost:
		var payload model.GenesisModelHashRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}
		if err := h.svc.UpsertGenesisModelHash(r.Context(), peer, payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]string{"jobId": payload.JobID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var errJobIDRequired = errors.New("jobId query parameter is required")
