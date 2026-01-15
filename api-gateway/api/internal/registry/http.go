package registry

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/nebula/api-gateway/internal/common"
)

// HTTPHandler exposes registry endpoints.
type HTTPHandler struct {
	svc *Service
}

// NewHTTPHandler wires a registry HTTP handler.
func NewHTTPHandler(svc *Service) *HTTPHandler {
	return &HTTPHandler{svc: svc}
}

// RegisterRoutes mounts the enrollment endpoint.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux, auth *common.Authenticator) {
	mux.Handle("/auth/register-trainer", auth.RequireAuth(http.HandlerFunc(h.handleRegister)))
	mux.Handle("/auth/register-trainers", auth.RequireAuth(http.HandlerFunc(h.handleBulkRegister), common.RoleAdmin))
}

type registerRequest struct {
	DID             string          `json:"did"`
	NodeID          string          `json:"nodeId"`
	VC              json.RawMessage `json:"vc"`
	PublicKey       string          `json:"public_key"`
	PublicKey2      string          `json:"publicKey"`
	JWTSubject      string          `json:"jwt_sub"`
	SubjectOverride string          `json:"subject"`
	State           string          `json:"state"`
	StateID         string          `json:"state_id"`
	Cluster         string          `json:"cluster"`
	ClusterID       string          `json:"cluster_id"`
}

func (r *registerRequest) toInput() RegisterInput {
	key := r.PublicKey
	if key == "" {
		key = r.PublicKey2
	}
	return RegisterInput{
		DID:        r.DID,
		NodeID:     r.NodeID,
		State:      r.stateValue(),
		Cluster:    r.clusterValue(),
		VC:         r.VC,
		PublicKey:  key,
		JWTSubject: r.requestedSubject(),
	}
}

func (r *registerRequest) requestedSubject() string {
	if sub := strings.TrimSpace(r.JWTSubject); sub != "" {
		return sub
	}
	if sub := strings.TrimSpace(r.SubjectOverride); sub != "" {
		return sub
	}
	return ""
}

func (r *registerRequest) fallbackSubject() string {
	if sub := r.requestedSubject(); sub != "" {
		return sub
	}
	if sub := strings.TrimSpace(r.NodeID); sub != "" {
		return sub
	}
	if sub := strings.TrimSpace(r.DID); sub != "" {
		return sub
	}
	return ""
}

func (r *registerRequest) stateValue() string {
	if val := strings.TrimSpace(r.StateID); val != "" {
		return val
	}
	if val := strings.TrimSpace(r.State); val != "" {
		return val
	}
	return ""
}

func (r *registerRequest) clusterValue() string {
	if val := strings.TrimSpace(r.ClusterID); val != "" {
		return val
	}
	if val := strings.TrimSpace(r.Cluster); val != "" {
		return val
	}
	return ""
}

func (h *HTTPHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	var payload registerRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	record, err := h.svc.Register(r.Context(), authCtx, payload.toInput())
	if err != nil {
		status := http.StatusInternalServerError
		if se, ok := common.AsStatusError(err); ok {
			status = se.Code
		}
		common.WriteErrorWithCode(w, status, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]any{
		"status":           "ok",
		"jwt_sub":          record.JWTSub,
		"fabric_client_id": record.FabricClientID,
		"vc_hash":          record.VCHash,
		"did":              record.DID,
		"node_id":          record.NodeID,
		"state":            record.State,
		"cluster":          record.Cluster,
		"registered_at":    record.RegisteredAt,
	})
}

type bulkRegisterResult struct {
	DID            string `json:"did"`
	NodeID         string `json:"nodeId"`
	JWTSub         string `json:"jwt_sub"`
	State          string `json:"state,omitempty"`
	Cluster        string `json:"cluster,omitempty"`
	Status         string `json:"status"`
	Error          string `json:"error,omitempty"`
	HTTPStatus     int    `json:"status_code,omitempty"`
	FabricClientID string `json:"fabric_client_id,omitempty"`
	VCHash         string `json:"vc_hash,omitempty"`
	RegisteredAt   string `json:"registered_at,omitempty"`
}

func (h *HTTPHandler) handleBulkRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	var payloads []registerRequest
	if err := json.NewDecoder(r.Body).Decode(&payloads); err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	if len(payloads) == 0 {
		common.WriteErrorWithCode(w, http.StatusBadRequest, common.NewStatusError(http.StatusBadRequest, "request body must contain at least one entry"))
		return
	}
	_, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	results := make([]bulkRegisterResult, 0, len(payloads))
	hasError := false
	for _, payload := range payloads {
		input := payload.toInput()
		if input.JWTSubject == "" {
			input.JWTSubject = payload.fallbackSubject()
		}
		subject := strings.TrimSpace(input.JWTSubject)
		if subject == "" {
			hasError = true
			results = append(results, bulkRegisterResult{
				DID:        payload.DID,
				NodeID:     payload.NodeID,
				Status:     "error",
				Error:      "subject could not be determined for this entry",
				HTTPStatus: http.StatusBadRequest,
			})
			continue
		}
		authCtx := &common.AuthContext{Subject: subject}
		record, err := h.svc.Register(r.Context(), authCtx, input)
		if err != nil {
			hasError = true
			status := http.StatusInternalServerError
			msg := err.Error()
			if se, ok := common.AsStatusError(err); ok {
				status = se.Code
				msg = se.Msg
			}
			results = append(results, bulkRegisterResult{
				DID:        payload.DID,
				NodeID:     payload.NodeID,
				JWTSub:     subject,
				Status:     "error",
				Error:      msg,
				HTTPStatus: status,
			})
			continue
		}
		results = append(results, bulkRegisterResult{
			DID:            record.DID,
			NodeID:         record.NodeID,
			JWTSub:         record.JWTSub,
			State:          record.State,
			Cluster:        record.Cluster,
			Status:         "ok",
			FabricClientID: record.FabricClientID,
			VCHash:         record.VCHash,
			RegisteredAt:   record.RegisteredAt,
		})
	}
	code := http.StatusOK
	if hasError {
		code = http.StatusMultiStatus
	}
	common.WriteJSON(w, code, map[string]any{"results": results})
}
