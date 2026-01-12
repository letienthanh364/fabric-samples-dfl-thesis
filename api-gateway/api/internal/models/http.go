package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/nebula/api-gateway/internal/common"
	"github.com/nebula/api-gateway/internal/registry"
)

// HTTPHandler exposes the scoped /models endpoints.
type HTTPHandler struct {
	svc   *Service
	store *registry.Store
}

// NewHTTPHandler prepares a HTTP handler.
func NewHTTPHandler(svc *Service, store *registry.Store) *HTTPHandler {
	return &HTTPHandler{svc: svc, store: store}
}

// RegisterRoutes wires the models endpoints for each configured layer.
func (h *HTTPHandler) RegisterRoutes(mux *http.ServeMux, auth *common.Authenticator) {
	keyFunc := func(header *common.TokenHeader, claims *common.JWTClaims) (*common.KeySpec, error) {
		subject := strings.TrimSpace(claims.Subject)
		if subject == "" {
			return nil, errors.New("token missing subject")
		}
		record, ok := h.store.FindByJWTSub(subject)
		if !ok {
			return nil, errors.New("trainer not registered")
		}
		pub, err := record.PublicKeyBytes()
		if err != nil {
			return nil, err
		}
		return &common.KeySpec{Algorithm: "EdDSA", PublicKey: pub}, nil
	}
	for _, layer := range h.svc.Layers() {
		if layer == nil {
			continue
		}
		layer := layer
		basePath := fmt.Sprintf("/%s/models", layer.Slug)
		mux.Handle(basePath, auth.RequireAuthWithKeyFunc(keyFunc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.handleCollection(w, r, layer)
		})))
		mux.Handle(basePath+"/", auth.RequireAuthWithKeyFunc(keyFunc, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.handleRecord(w, r, layer)
		})))
	}
}

func (h *HTTPHandler) handleCollection(w http.ResponseWriter, r *http.Request, layer *Layer) {
	switch r.Method {
	case http.MethodPost:
		h.handleCommit(w, r, layer)
	case http.MethodGet:
		h.handleList(w, r, layer)
	default:
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
	}
}

func (h *HTTPHandler) handleRecord(w http.ResponseWriter, r *http.Request, layer *Layer) {
	if r.Method != http.MethodGet {
		common.WriteErrorWithCode(w, http.StatusMethodNotAllowed, common.ErrMethodNotAllowed)
		return
	}
	prefix := fmt.Sprintf("/%s/models/", layer.Slug)
	dataID := strings.TrimPrefix(r.URL.Path, prefix)
	if dataID == "" {
		common.WriteErrorWithCode(w, http.StatusBadRequest, common.NewStatusError(http.StatusBadRequest, "data identifier missing"))
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	record, err := h.svc.Retrieve(r.Context(), authCtx, dataID)
	if err != nil {
		status := http.StatusInternalServerError
		if se, ok := common.AsStatusError(err); ok {
			status = se.Code
		}
		common.WriteErrorWithCode(w, status, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, record)
}

func (h *HTTPHandler) handleCommit(w http.ResponseWriter, r *http.Request, layer *Layer) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	payload := body["payload"]
	if len(payload) == 0 {
		common.WriteErrorWithCode(w, http.StatusBadRequest, common.NewStatusError(http.StatusBadRequest, "payload is required"))
		return
	}
	scopeID, err := extractScopeID(body, layer)
	if err != nil {
		common.WriteErrorWithCode(w, http.StatusBadRequest, err)
		return
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	result, err := h.svc.Commit(r.Context(), authCtx, layer.Slug, scopeID, payload)
	if err != nil {
		status := http.StatusInternalServerError
		if se, ok := common.AsStatusError(err); ok {
			status = se.Code
		}
		common.WriteErrorWithCode(w, status, err)
		return
	}
	common.WriteJSON(w, http.StatusCreated, result)
}

func (h *HTTPHandler) handleList(w http.ResponseWriter, r *http.Request, layer *Layer) {
	query := r.URL.Query()
	scopeID := strings.TrimSpace(query.Get("scopeId"))
	if scopeID == "" {
		scopeID = strings.TrimSpace(query.Get("scope_id"))
	}
	page := 1
	if raw := strings.TrimSpace(query.Get("page")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 {
			common.WriteErrorWithCode(w, http.StatusBadRequest, common.NewStatusError(http.StatusBadRequest, "page must be a positive integer"))
			return
		}
		page = value
	}
	authCtx, ok := common.AuthContextFrom(r.Context())
	if !ok {
		common.WriteErrorWithCode(w, http.StatusUnauthorized, common.ErrMissingAuthContext)
		return
	}
	result, err := h.svc.List(r.Context(), authCtx, layer.Slug, scopeID, page)
	if err != nil {
		status := http.StatusInternalServerError
		if se, ok := common.AsStatusError(err); ok {
			status = se.Code
		}
		common.WriteErrorWithCode(w, status, err)
		return
	}
	common.WriteJSON(w, http.StatusOK, result)
}

func extractScopeID(body map[string]json.RawMessage, layer *Layer) (string, error) {
	candidates := []string{layer.ScopeField, "scope_id", "scopeId"}
	for _, key := range candidates {
		if key == "" {
			continue
		}
		raw, ok := body[key]
		if !ok {
			continue
		}
		var scope string
		if err := json.Unmarshal(raw, &scope); err != nil {
			return "", common.NewStatusError(http.StatusBadRequest, fmt.Sprintf("%s must be a string", key))
		}
		scope = strings.TrimSpace(scope)
		if scope != "" {
			return scope, nil
		}
	}
	return "", nil
}
