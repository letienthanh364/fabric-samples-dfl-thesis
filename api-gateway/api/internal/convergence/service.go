package convergence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/nebula/api-gateway/internal/common"
	"github.com/nebula/api-gateway/internal/registry"
	"github.com/nebula/api-gateway/internal/whitelist"
)

// Service coordinates convergence operations.
type Service struct {
	cfg       *common.Config
	fabric    *common.FabricClient
	store     *registry.Store
	whitelist *whitelist.Service
}

// NewService creates a convergence service.
func NewService(cfg *common.Config, fabric *common.FabricClient, store *registry.Store, whitelist *whitelist.Service) *Service {
	return &Service{cfg: cfg, fabric: fabric, store: store, whitelist: whitelist}
}

// CommitRequest captures convergence payloads submitted by aggregators.
type CommitRequest struct {
	StateID   string         `json:"state_id"`
	ClusterID string         `json:"cluster_id,omitempty"`
	Payload   map[string]any `json:"payload"`
}

// DeclareRequest captures "all converged" submissions.
type DeclareRequest struct {
	StateID string         `json:"state_id,omitempty"`
	Payload map[string]any `json:"payload"`
}

// ClusterStatus describes the convergence state for a cluster.
type ClusterStatus struct {
	ClusterID   string         `json:"cluster_id"`
	IsConverged bool           `json:"is_converged"`
	SubmittedAt string         `json:"submitted_at,omitempty"`
	SourceID    string         `json:"source_id,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// StateStatus summarizes convergence for a state.
type StateStatus struct {
	StateID        string           `json:"state_id"`
	IsConverged    bool             `json:"is_converged"`
	ConvergedAt    string           `json:"converged_at,omitempty"`
	DeclaredBy     string           `json:"declared_by,omitempty"`
	SummaryPayload map[string]any   `json:"summary_payload,omitempty"`
	Clusters       []*ClusterStatus `json:"clusters"`
}

// NationStatus summarizes convergence for the nation.
type NationStatus struct {
	IsConverged    bool              `json:"is_converged"`
	ConvergedAt    string            `json:"converged_at,omitempty"`
	DeclaredBy     string            `json:"declared_by,omitempty"`
	SummaryPayload map[string]any    `json:"summary_payload,omitempty"`
	States         []*StateAggregate `json:"states"`
}

// StateAggregate captures nation-level convergence per state.
type StateAggregate struct {
	StateID     string         `json:"state_id"`
	IsConverged bool           `json:"is_converged"`
	SubmittedAt string         `json:"submitted_at,omitempty"`
	SourceID    string         `json:"source_id,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

// CommitStateCluster records a cluster -> state convergence payload.
func (s *Service) CommitStateCluster(ctx context.Context, authCtx *common.AuthContext, req *CommitRequest) error {
	if authCtx == nil {
		return common.NewStatusError(http.StatusUnauthorized, "authentication context missing")
	}
	if req == nil {
		return common.NewStatusError(http.StatusBadRequest, "request body is required")
	}
	stateID := selectValue(req.StateID, authCtx.State)
	if strings.TrimSpace(stateID) == "" {
		return common.NewStatusError(http.StatusBadRequest, "state_id is required")
	}
	clusterID := selectValue(req.ClusterID, authCtx.Cluster)
	if strings.TrimSpace(clusterID) == "" {
		return common.NewStatusError(http.StatusBadRequest, "cluster_id is required")
	}
	payload, err := marshalPayload(req.Payload)
	if err != nil {
		return err
	}
	rec, ok := s.store.FindByJWTSub(authCtx.Subject)
	if !ok {
		return common.NewStatusError(http.StatusForbidden, "trainer not registered")
	}
	args := []string{"CommitStateClusterConvergence", stateID, clusterID, payload}
	return s.invoke(authCtx, rec.FabricClientID, args)
}

// CommitNationState records a state -> nation convergence payload.
func (s *Service) CommitNationState(ctx context.Context, authCtx *common.AuthContext, req *CommitRequest) error {
	if authCtx == nil {
		return common.NewStatusError(http.StatusUnauthorized, "authentication context missing")
	}
	if req == nil {
		return common.NewStatusError(http.StatusBadRequest, "request body is required")
	}
	stateID := selectValue(req.StateID, authCtx.State)
	if strings.TrimSpace(stateID) == "" {
		return common.NewStatusError(http.StatusBadRequest, "state_id is required")
	}
	payload, err := marshalPayload(req.Payload)
	if err != nil {
		return err
	}
	rec, ok := s.store.FindByJWTSub(authCtx.Subject)
	if !ok {
		return common.NewStatusError(http.StatusForbidden, "trainer not registered")
	}
	args := []string{"CommitNationStateConvergence", stateID, payload}
	return s.invoke(authCtx, rec.FabricClientID, args)
}

// DeclareStateAll records that all clusters in a state are converged.
func (s *Service) DeclareStateAll(ctx context.Context, authCtx *common.AuthContext, req *DeclareRequest) error {
	if authCtx == nil {
		return common.NewStatusError(http.StatusUnauthorized, "authentication context missing")
	}
	if req == nil {
		return common.NewStatusError(http.StatusBadRequest, "request body is required")
	}
	stateID := selectValue(req.StateID, authCtx.State)
	if strings.TrimSpace(stateID) == "" {
		return common.NewStatusError(http.StatusBadRequest, "state_id is required")
	}
	payload, err := marshalPayload(req.Payload)
	if err != nil {
		return err
	}
	rec, ok := s.store.FindByJWTSub(authCtx.Subject)
	if !ok {
		return common.NewStatusError(http.StatusForbidden, "trainer not registered")
	}
	args := []string{"DeclareStateConvergence", stateID, payload}
	return s.invoke(authCtx, rec.FabricClientID, args)
}

// DeclareNationAll records that all states are converged at the nation scope.
func (s *Service) DeclareNationAll(ctx context.Context, authCtx *common.AuthContext, req *DeclareRequest) error {
	if authCtx == nil {
		return common.NewStatusError(http.StatusUnauthorized, "authentication context missing")
	}
	payload, err := marshalPayload(req.Payload)
	if err != nil {
		return err
	}
	rec, ok := s.store.FindByJWTSub(authCtx.Subject)
	if !ok {
		return common.NewStatusError(http.StatusForbidden, "trainer not registered")
	}
	args := []string{"DeclareNationConvergence", payload}
	return s.invoke(authCtx, rec.FabricClientID, args)
}

// StateStatus resolves convergence for a state.
func (s *Service) StateStatus(ctx context.Context, authCtx *common.AuthContext, stateID string) (*StateStatus, error) {
	if authCtx != nil {
		stateID = selectValue(stateID, authCtx.State)
	}
	if strings.TrimSpace(stateID) == "" {
		return nil, common.NewStatusError(http.StatusBadRequest, "state_id is required")
	}
	identity, err := s.identityFor(authCtx)
	if err != nil {
		return nil, err
	}
	args := []string{"ReadStateConvergence", stateID}
	payload, err := s.fabric.QueryChaincode(s.fabric.SelectPeer(), identity, args)
	if err != nil {
		return nil, err
	}
	var ledgerState ledgerStateConvergence
	if err := json.Unmarshal(payload, &ledgerState); err != nil {
		return nil, err
	}
	return s.stateStatusFromLedger(ctx, &ledgerState)
}

// NationStatus resolves convergence for the nation.
func (s *Service) NationStatus(ctx context.Context, authCtx *common.AuthContext) (*NationStatus, error) {
	identity, err := s.identityFor(authCtx)
	if err != nil {
		return nil, err
	}
	args := []string{"ReadNationConvergence"}
	payload, err := s.fabric.QueryChaincode(s.fabric.SelectPeer(), identity, args)
	if err != nil {
		return nil, err
	}
	var ledgerNation ledgerNationConvergence
	if err := json.Unmarshal(payload, &ledgerNation); err != nil {
		return nil, err
	}
	return s.nationStatusFromLedger(ctx, &ledgerNation)
}

// ListStateStatuses returns convergence data for all states (admin only).
func (s *Service) ListStateStatuses(ctx context.Context, authCtx *common.AuthContext) (map[string]*StateStatus, error) {
	identity, err := s.identityFor(authCtx)
	if err != nil {
		return nil, err
	}
	args := []string{"ListStateConvergence"}
	payload, err := s.fabric.QueryChaincode(s.fabric.SelectPeer(), identity, args)
	if err != nil {
		return nil, err
	}
	var raw map[string]*ledgerStateConvergence
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	results := make(map[string]*StateStatus, len(raw))
	for stateID, entry := range raw {
		entry.StateID = stateID
		status, err := s.stateStatusFromLedger(ctx, entry)
		if err != nil {
			return nil, err
		}
		results[stateID] = status
	}
	return results, nil
}

// ListNationStatus returns the detailed nation convergence map.
func (s *Service) ListNationStatus(ctx context.Context, authCtx *common.AuthContext) (*NationStatus, error) {
	return s.NationStatus(ctx, authCtx)
}

func (s *Service) invoke(authCtx *common.AuthContext, identity string, args []string) error {
	peer := s.fabric.SelectPeer()
	if peer == "" {
		return common.NewStatusError(http.StatusInternalServerError, "no fabric peers configured")
	}
	return s.fabric.InvokeChaincode(peer, identity, args)
}

func (s *Service) identityFor(authCtx *common.AuthContext) (string, error) {
	if authCtx != nil {
		if rec, ok := s.store.FindByJWTSub(authCtx.Subject); ok {
			return rec.FabricClientID, nil
		}
	}
	if authCtx != nil && authCtx.Role == common.RoleAdmin {
		return s.cfg.AdminIdentity, nil
	}
	return s.cfg.AdminIdentity, nil
}

func marshalPayload(payload map[string]any) (string, error) {
	if len(payload) == 0 {
		return "", common.NewStatusError(http.StatusBadRequest, "payload is required")
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func selectValue(values ...string) string {
	for _, val := range values {
		if strings.TrimSpace(val) != "" {
			return val
		}
	}
	return ""
}

func (s *Service) stateStatusFromLedger(ctx context.Context, entry *ledgerStateConvergence) (*StateStatus, error) {
	if entry == nil {
		return nil, errors.New("state convergence data missing")
	}
	stateID := entry.StateID
	if stateID == "" {
		stateID = entry.ID()
	}
	clusters, err := s.clustersForState(ctx, stateID)
	if err != nil {
		return nil, err
	}
	status := &StateStatus{
		StateID:  stateID,
		Clusters: make([]*ClusterStatus, 0, len(clusters)),
	}
	clusterMap := entry.Clusters
	for _, clusterID := range clusters {
		record := clusterMap[clusterID]
		clusterStatus := &ClusterStatus{ClusterID: clusterID}
		if record != nil {
			clusterStatus.IsConverged = true
			clusterStatus.SubmittedAt = record.SubmittedAt
			clusterStatus.SourceID = record.SourceID
			clusterStatus.Payload = decodePayload(record.Payload)
		}
		status.Clusters = append(status.Clusters, clusterStatus)
	}
	if entry.Summary != nil {
		status.IsConverged = true
		status.ConvergedAt = entry.Summary.DeclaredAt
		status.DeclaredBy = entry.Summary.DeclaredBy
		status.SummaryPayload = decodePayload(entry.Summary.Payload)
	} else {
		allConverged := true
		for _, cluster := range status.Clusters {
			if !cluster.IsConverged {
				allConverged = false
				break
			}
		}
		status.IsConverged = allConverged && len(status.Clusters) > 0
		if status.IsConverged {
			status.ConvergedAt = latestClusterTime(status.Clusters)
		}
	}
	return status, nil
}

func (s *Service) nationStatusFromLedger(ctx context.Context, entry *ledgerNationConvergence) (*NationStatus, error) {
	if entry == nil {
		return &NationStatus{}, nil
	}
	hierarchy, err := s.whitelist.Hierarchy(ctx)
	if err != nil {
		return nil, err
	}
	stateIDs := hierarchyStateIDs(hierarchy)
	states := make([]*StateAggregate, 0, len(stateIDs))
	allConverged := true
	var latest string
	for _, stateID := range stateIDs {
		record := entry.States[stateID]
		stateAggregate := &StateAggregate{StateID: stateID}
		if record != nil {
			stateAggregate.IsConverged = true
			stateAggregate.SubmittedAt = record.SubmittedAt
			stateAggregate.SourceID = record.SourceID
			stateAggregate.Payload = decodePayload(record.Payload)
			if record.SubmittedAt > latest {
				latest = record.SubmittedAt
			}
		} else {
			allConverged = false
		}
		states = append(states, stateAggregate)
	}
	status := &NationStatus{
		States: states,
	}
	if entry.Summary != nil {
		status.IsConverged = true
		status.ConvergedAt = entry.Summary.DeclaredAt
		status.DeclaredBy = entry.Summary.DeclaredBy
		status.SummaryPayload = decodePayload(entry.Summary.Payload)
	} else {
		status.IsConverged = allConverged && len(states) > 0
		if status.IsConverged {
			status.ConvergedAt = latest
		}
	}
	return status, nil
}

func (s *Service) clustersForState(ctx context.Context, stateID string) ([]string, error) {
	hierarchy, err := s.whitelist.Hierarchy(ctx)
	if err != nil {
		return nil, err
	}
	for _, state := range hierarchy.States {
		if state == nil {
			continue
		}
		if strings.EqualFold(state.StateID, stateID) {
			ids := make([]string, 0, len(state.Clusters))
			for _, cluster := range state.Clusters {
				if cluster == nil {
					continue
				}
				ids = append(ids, cluster.ClusterID)
			}
			sort.Strings(ids)
			return ids, nil
		}
	}
	return nil, common.NewStatusError(http.StatusBadRequest, fmt.Sprintf("state %s not found in whitelist", stateID))
}

func decodePayload(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{"raw": string(raw)}
	}
	return out
}

func latestClusterTime(clusters []*ClusterStatus) string {
	latest := ""
	for _, c := range clusters {
		if c != nil && c.SubmittedAt > latest {
			latest = c.SubmittedAt
		}
	}
	return latest
}

type ledgerConvergenceRecord struct {
	Scope       string          `json:"scope"`
	StateID     string          `json:"state_id"`
	ClusterID   string          `json:"cluster_id"`
	SourceID    string          `json:"source_id"`
	Payload     json.RawMessage `json:"payload"`
	SubmittedAt string          `json:"submitted_at"`
}

type ledgerConvergenceSummary struct {
	Scope      string          `json:"scope"`
	TargetID   string          `json:"target_id"`
	DeclaredBy string          `json:"declared_by"`
	DeclaredAt string          `json:"declared_at"`
	Payload    json.RawMessage `json:"payload"`
}

type ledgerStateConvergence struct {
	StateID  string                              `json:"state_id"`
	Clusters map[string]*ledgerConvergenceRecord `json:"clusters"`
	Summary  *ledgerConvergenceSummary           `json:"summary"`
}

func (s *ledgerStateConvergence) ID() string {
	if s == nil {
		return ""
	}
	return s.StateID
}

type ledgerNationConvergence struct {
	States  map[string]*ledgerConvergenceRecord `json:"states"`
	Summary *ledgerConvergenceSummary           `json:"summary"`
}

func hierarchyStateIDs(h *whitelist.HierarchyResult) []string {
	if h == nil {
		return nil
	}
	ids := make([]string, 0, len(h.States))
	for _, state := range h.States {
		if state == nil {
			continue
		}
		ids = append(ids, state.StateID)
	}
	sort.Strings(ids)
	return ids
}
