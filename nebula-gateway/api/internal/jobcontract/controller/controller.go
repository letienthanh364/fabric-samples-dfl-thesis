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
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *common.Authenticator) {
	mux.Handle("/job-contract/genesis-model-cid", auth.RequireAuth(http.HandlerFunc(h.handleGenesisModelCID)))
	mux.Handle("/job-contract/genesis-model-hash", auth.RequireAuth(http.HandlerFunc(h.handleGenesisModelHash)))
	mux.Handle("/job-contract/training-config", auth.RequireAuth(http.HandlerFunc(h.handleTrainingConfig)))
}

func (h *Handler) handleGenesisModelCID(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, errors.New("authentication context missing"))
		return
	}
	peer, err := h.cfg.PeerForState(authCtx.State)
	if err != nil {
		common.WriteErrorWithCode(w, http.StatusForbidden, err)
		return
	}
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
		if authCtx.Role != common.RoleAdmin {
			common.WriteErrorWithCode(w, http.StatusForbidden, errors.New("admin role required"))
			return
		}
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
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, errors.New("authentication context missing"))
		return
	}
	peer, err := h.cfg.PeerForState(authCtx.State)
	if err != nil {
		common.WriteErrorWithCode(w, http.StatusForbidden, err)
		return
	}
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
		if authCtx.Role != common.RoleAdmin {
			common.WriteErrorWithCode(w, http.StatusForbidden, errors.New("admin role required"))
			return
		}
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

func (h *Handler) handleTrainingConfig(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, errors.New("authentication context missing"))
		return
	}
	peer, err := h.cfg.PeerForState(authCtx.State)
	if err != nil {
		common.WriteErrorWithCode(w, http.StatusForbidden, err)
		return
	}
	switch r.Method {
	case http.MethodGet:
		jobID := strings.TrimSpace(r.URL.Query().Get("jobId"))
		if jobID == "" {
			common.WriteErrorWithCode(w, http.StatusBadRequest, errJobIDRequired)
			return
		}
		record, err := h.svc.GetTrainingConfig(r.Context(), peer, jobID)
		if err != nil {
			common.WriteError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, record)
	case http.MethodPost:
		if authCtx.Role != common.RoleAdmin {
			common.WriteErrorWithCode(w, http.StatusForbidden, errors.New("admin role required"))
			return
		}
		var payload model.TrainingConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, fmt.Errorf("invalid payload: %w", err))
			return
		}
		if err := h.svc.UpsertTrainingConfig(r.Context(), peer, payload); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]string{"jobId": payload.JobID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var errJobIDRequired = errors.New("jobId query parameter is required")
