package convergence

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nebula/api-gateway/internal/common"
)

// HTTPHandler wires convergence routes.
type HTTPHandler struct {
	svc *Service
}

// NewHTTPHandler creates a convergence HTTP handler.
func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// RegisterRoutes adds convergence endpoints to the mux.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux, auth *common.Authenticator) {
	mux.Handle("/state/convergence", auth.RequireAuth(http.HandlerFunc(h.handleStateConvergence), common.RoleTrainer, common.RoleAggregator, common.RoleCentralChecker, common.RoleAdmin))
	mux.Handle("/state/convergence/all", auth.RequireAuth(http.HandlerFunc(h.handleStateAll), common.RoleCentralChecker))
	mux.Handle("/state/convergence/list", auth.RequireAuth(http.HandlerFunc(h.handleStateList), common.RoleAdmin))

	mux.Handle("/nation/convergence", auth.RequireAuth(http.HandlerFunc(h.handleNationConvergence), common.RoleTrainer, common.RoleAggregator, common.RoleCentralChecker, common.RoleAdmin))
	mux.Handle("/nation/convergence/all", auth.RequireAuth(http.HandlerFunc(h.handleNationAll), common.RoleCentralChecker))
	mux.Handle("/nation/convergence/list", auth.RequireAuth(http.HandlerFunc(h.handleNationList), common.RoleAdmin))
}

func (h *HTTPHandler) handleStateConvergence(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	switch r.Method {
	case http.MethodPost:
		if authCtx.Role != common.RoleAggregator {
			common.WriteErrorWithCode(w, http.StatusForbidden, common.NewStatusError(http.StatusForbidden, "only aggregators can submit convergence payloads"))
			return
		}
		var req CommitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		if err := h.svc.CommitStateCluster(r.Context(), authCtx, &req); err != nil {
			writeServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
	case http.MethodGet:
		stateID := strings.TrimSpace(r.URL.Query().Get("stateId"))
		status, err := h.svc.StateStatus(r.Context(), authCtx, stateID)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, status)
	default:
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
	}
}

func (h *HTTPHandler) handleStateAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	var req DeclareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.DeclareStateAll(r.Context(), authCtx, &req); err != nil {
		writeServiceError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (h *HTTPHandler) handleStateList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	result, err := h.svc.ListStateStatuses(r.Context(), authCtx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func (h *HTTPHandler) handleNationConvergence(w http.ResponseWriter, r *http.Request) {
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	switch r.Method {
	case http.MethodPost:
		if authCtx.Role != common.RoleAggregator {
			common.WriteErrorWithCode(w, http.StatusForbidden, common.NewStatusError(http.StatusForbidden, "only aggregators can submit convergence payloads"))
			return
		}
		var req CommitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			common.WriteErrorWithCode(w, http.StatusBadRequest, err)
			return
		}
		if err := h.svc.CommitNationState(r.Context(), authCtx, &req); err != nil {
			writeServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
	case http.MethodGet:
		status, err := h.svc.NationStatus(r.Context(), authCtx)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		common.WriteJSON(w, http.StatusOK, status)
	default:
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
	}
}

func (h *HTTPHandler) handleNationAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	var req DeclareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	if err := h.svc.DeclareNationAll(r.Context(), authCtx, &req); err != nil {
		writeServiceError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]any{"status": "ok"})
}

func (h *HTTPHandler) handleNationList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	result, err := h.svc.ListNationStatus(r.Context(), authCtx)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func writeServiceError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if se, ok := common.AsStatusError(err); ok {
		status = se.Code
	}
	common.WriteErrorWithCode(w, status, err)
}
