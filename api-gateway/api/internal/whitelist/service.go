package whitelist

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/nebula/api-gateway/internal/common"
)

const defaultPageSize = 50

// Service exposes helper methods to fetch the Fabric whitelist.
type Service struct {
	cfg    *common.Config
	fabric *common.FabricClient
}

// Entry describes a trainer record.
type Entry struct {
	JWTSub       string `json:"jwt_sub"`
	DID          string `json:"did"`
	NodeID       string `json:"node_id"`
	State        string `json:"state,omitempty"`
	Cluster      string `json:"cluster,omitempty"`
	VCHash       string `json:"vc_hash"`
	PublicKey    string `json:"public_key"`
	RegisteredAt string `json:"registered_at"`
}

// ListResult represents a page of whitelist entries.
type ListResult struct {
	Items   []*Entry `json:"items"`
	Page    int      `json:"page"`
	PerPage int      `json:"per_page"`
	Total   int      `json:"total"`
	HasMore bool     `json:"has_more"`
}

// HierarchyResult represents the whitelist grouped by state/cluster.
type HierarchyResult struct {
	States  []*StateGroup `json:"states"`
	Page    int           `json:"page"`
	PerPage int           `json:"per_page"`
	Total   int           `json:"total"`
	HasMore bool          `json:"has_more"`
}

// StateGroup captures clusters per state.
type StateGroup struct {
	StateID  string          `json:"state_id"`
	Clusters []*ClusterGroup `json:"clusters"`
}

// ClusterGroup lists nodes in a cluster.
type ClusterGroup struct {
	ClusterID string   `json:"cluster_id"`
	Nodes     []*Entry `json:"nodes"`
}

// NewService constructs a whitelist service instance.
func NewService(cfg *common.Config, fabric *common.FabricClient) *Service {
	return &Service{cfg: cfg, fabric: fabric}
}

// Hierarchy fetches the entire whitelist hierarchy.
func (s *Service) Hierarchy(ctx context.Context) (*HierarchyResult, error) {
	page := 1
	all := make([]*Entry, 0)
	for {
		result, err := s.List(ctx, page, defaultPageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, result.Items...)
		if !result.HasMore {
			break
		}
		page++
	}
	combined := &ListResult{
		Items:   all,
		Page:    1,
		PerPage: len(all),
		Total:   len(all),
		HasMore: false,
	}
	return combined.ToHierarchy(), nil
}

// List returns whitelist entries from the Fabric ledger.
func (s *Service) List(ctx context.Context, page, perPage int) (*ListResult, error) {
	if page < 1 {
		return nil, common.NewStatusError(http.StatusBadRequest, "page must be >= 1")
	}
	if perPage < 1 {
		perPage = defaultPageSize
	}
	peerName := s.fabric.SelectPeer()
	if peerName == "" {
		return nil, common.NewStatusError(http.StatusInternalServerError, "no fabric peers configured")
	}
	args := []string{
		"ListWhitelist",
		strconv.Itoa(page),
		strconv.Itoa(perPage),
	}
	raw, err := s.fabric.QueryChaincode(peerName, s.cfg.AdminIdentity, args)
	if err != nil {
		return nil, err
	}
	var ledgerPage ledgerList
	if err := json.Unmarshal(raw, &ledgerPage); err != nil {
		return nil, err
	}
	return ledgerPage.toResult(), nil
}

type ledgerEntry struct {
	JWTSub     string `json:"jwt_sub"`
	DID        string `json:"did"`
	NodeID     string `json:"node_id"`
	State      string `json:"state,omitempty"`
	Cluster    string `json:"cluster,omitempty"`
	VCHash     string `json:"vc_hash"`
	PublicKey  string `json:"public_key"`
	Registered string `json:"registered_at"`
}

type ledgerList struct {
	Items   []*ledgerEntry `json:"items"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
	Total   int            `json:"total"`
	HasMore bool           `json:"has_more"`
}

func (l *ledgerList) toResult() *ListResult {
	result := &ListResult{
		Page:    l.Page,
		PerPage: l.PerPage,
		Total:   l.Total,
		HasMore: l.HasMore,
	}
	if len(l.Items) == 0 {
		return result
	}
	items := make([]*Entry, 0, len(l.Items))
	for _, entry := range l.Items {
		if entry == nil {
			continue
		}
		items = append(items, &Entry{
			JWTSub:       entry.JWTSub,
			DID:          entry.DID,
			NodeID:       entry.NodeID,
			State:        entry.State,
			Cluster:      entry.Cluster,
			VCHash:       entry.VCHash,
			PublicKey:    entry.PublicKey,
			RegisteredAt: entry.Registered,
		})
	}
	result.Items = items
	return result
}

// ToHierarchy groups entries by state and cluster.
func (r *ListResult) ToHierarchy() *HierarchyResult {
	hierarchy := &HierarchyResult{
		Page:    r.Page,
		PerPage: r.PerPage,
		Total:   r.Total,
		HasMore: r.HasMore,
	}
	if len(r.Items) == 0 {
		return hierarchy
	}
	stateMap := make(map[string]map[string][]*Entry)
	for _, entry := range r.Items {
		if entry == nil {
			continue
		}
		stateID := strings.TrimSpace(entry.State)
		if stateID == "" {
			stateID = "unknown"
		}
		clusterID := strings.TrimSpace(entry.Cluster)
		if clusterID == "" {
			clusterID = "unassigned"
		}
		if _, ok := stateMap[stateID]; !ok {
			stateMap[stateID] = make(map[string][]*Entry)
		}
		stateMap[stateID][clusterID] = append(stateMap[stateID][clusterID], entry)
	}
	stateIDs := make([]string, 0, len(stateMap))
	for stateID := range stateMap {
		stateIDs = append(stateIDs, stateID)
	}
	sort.Strings(stateIDs)
	for _, stateID := range stateIDs {
		clusterMap := stateMap[stateID]
		clusterIDs := make([]string, 0, len(clusterMap))
		for clusterID := range clusterMap {
			clusterIDs = append(clusterIDs, clusterID)
		}
		sort.Strings(clusterIDs)
		clusterGroups := make([]*ClusterGroup, 0, len(clusterIDs))
		for _, clusterID := range clusterIDs {
			clusterGroups = append(clusterGroups, &ClusterGroup{
				ClusterID: clusterID,
				Nodes:     clusterMap[clusterID],
			})
		}
		hierarchy.States = append(hierarchy.States, &StateGroup{
			StateID:  stateID,
			Clusters: clusterGroups,
		})
	}
	return hierarchy
}
