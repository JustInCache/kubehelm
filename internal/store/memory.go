package store

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/ankushko/k8s-project-revamp/internal/types"
)

type MemoryRepository struct {
	mu               sync.RWMutex
	users            []types.User
	clusters         []types.Cluster
	releases         []types.HelmRelease
	history          []types.ReleaseRevision
	approvals        []types.ReleaseApproval
	driftItems       []types.DriftItem
	audit            []types.AuditEvent
	compliance       []types.ComplianceCheck
	channels         []types.NotificationChannel
	rules            []types.NotificationRule
	reports          []types.Report
	org              types.OrgSettings
	helmRepos        []types.HelmRepository
	enabledProviders []string
}

func NewMemoryRepository() *MemoryRepository {
	orgID := "00000000-0000-0000-0000-000000000001"
	return &MemoryRepository{
		users: []types.User{
			{
				ID:       "00000000-0000-0000-0000-000000000002",
				Email:    "admin@kubeaudit.io",
				Name:     "Admin User",
				Role:     "admin",
				OrgID:    orgID,
				Password: "Admin@123",
				Active:   true,
			},
		},
		clusters:    []types.Cluster{},
		releases:    []types.HelmRelease{},
		history:     []types.ReleaseRevision{},
		approvals:   []types.ReleaseApproval{},
		driftItems:  []types.DriftItem{},
		audit:       []types.AuditEvent{},
		compliance:  []types.ComplianceCheck{},
		channels:    []types.NotificationChannel{},
		rules:       []types.NotificationRule{},
		reports:     []types.Report{},
		org: types.OrgSettings{
			ID:   orgID,
			Name: "Default Organization",
			Settings: map[string]any{
				"theme": "dark",
			},
		},
	}
}

func (m *MemoryRepository) FindUserByEmail(_ context.Context, email string) (*types.User, error) {
	for _, u := range m.users {
		if strings.EqualFold(u.Email, email) && u.Active {
			uc := u
			return &uc, nil
		}
	}
	return nil, nil
}

func (m *MemoryRepository) FindUserByID(_ context.Context, id string) (*types.User, error) {
	for _, u := range m.users {
		if u.ID == id && u.Active {
			uc := u
			return &uc, nil
		}
	}
	return nil, nil
}

func (m *MemoryRepository) ListClusters(_ context.Context, orgID string, page, limit int) (types.Paginated[types.Cluster], error) {
	filtered := make([]types.Cluster, 0)
	for _, c := range m.clusters {
		if c.OrgID == orgID {
			filtered = append(filtered, c)
		}
	}
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) GetCluster(_ context.Context, orgID, clusterID string) (*types.Cluster, error) {
	for _, c := range m.clusters {
		if c.OrgID == orgID && c.ID == clusterID {
			cp := c
			cp.KubeconfigRaw = ""
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *MemoryRepository) GetClusterKubeconfig(_ context.Context, orgID, clusterID string) (string, error) {
	for _, c := range m.clusters {
		if c.OrgID == orgID && c.ID == clusterID {
			return c.KubeconfigRaw, nil
		}
	}
	return "", nil
}

func (m *MemoryRepository) CreateCluster(_ context.Context, c types.Cluster) (*types.Cluster, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	m.clusters = append(m.clusters, c)
	cp := c
	cp.KubeconfigRaw = ""
	return &cp, nil
}

func (m *MemoryRepository) UpdateClusterStatus(_ context.Context, clusterID, status, serverVersion, lastError string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.clusters {
		if c.ID == clusterID {
			m.clusters[i].Status = status
			if serverVersion != "" {
				m.clusters[i].ServerVersion = serverVersion
			}
			m.clusters[i].LastError = lastError
			return nil
		}
	}
	return fmt.Errorf("cluster not found")
}

func (m *MemoryRepository) DeleteCluster(_ context.Context, orgID, clusterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, c := range m.clusters {
		if c.OrgID == orgID && c.ID == clusterID {
			m.clusters = append(m.clusters[:i], m.clusters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("cluster not found")
}

func (m *MemoryRepository) CreateAuditEvent(_ context.Context, event types.AuditEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	m.audit = append([]types.AuditEvent{event}, m.audit...)
	return nil
}

func (m *MemoryRepository) ListReleases(_ context.Context, orgID, namespace string, page, limit int, search, sortBy, sortOrder string) (types.Paginated[types.HelmRelease], error) {
	_ = orgID
	filtered := make([]types.HelmRelease, 0)
	for _, r := range m.releases {
		if namespace != "" && namespace != "all" && r.Namespace != namespace {
			continue
		}
		if search != "" {
			s := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(r.Name), s) && !strings.Contains(strings.ToLower(r.ChartName), s) {
				continue
			}
		}
		filtered = append(filtered, r)
	}
	if sortBy == "" {
		sortBy = "createdAt"
	}
	if sortOrder == "" {
		sortOrder = "desc"
	}
	slices.SortStableFunc(filtered, func(a, b types.HelmRelease) int {
		sign := 1
		if strings.EqualFold(sortOrder, "desc") {
			sign = -1
		}
		switch sortBy {
		case "name":
			return sign * strings.Compare(a.Name, b.Name)
		case "namespace":
			return sign * strings.Compare(a.Namespace, b.Namespace)
		case "status":
			return sign * strings.Compare(a.Status, b.Status)
		default:
			if a.CreatedAt.Equal(b.CreatedAt) {
				return 0
			}
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1 * sign
			}
			return sign
		}
	})
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) ListReleaseHistory(_ context.Context, releaseID string, page, limit int) (types.Paginated[types.ReleaseRevision], error) {
	filtered := make([]types.ReleaseRevision, 0)
	for _, h := range m.history {
		if h.ReleaseID == releaseID {
			filtered = append(filtered, h)
		}
	}
	slices.SortStableFunc(filtered, func(a, b types.ReleaseRevision) int {
		if a.Revision == b.Revision {
			return 0
		}
		if a.Revision > b.Revision {
			return -1
		}
		return 1
	})
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) GetReleaseManifest(_ context.Context, releaseID string, revision int) (string, error) {
	for _, h := range m.history {
		if h.ReleaseID == releaseID && h.Revision == revision {
			return h.Manifest, nil
		}
	}
	return "", fmt.Errorf("revision not found")
}

func (m *MemoryRepository) ListApprovals(_ context.Context, orgID, status string, page, limit int) (types.Paginated[types.ReleaseApproval], error) {
	_ = orgID
	filtered := make([]types.ReleaseApproval, 0)
	for _, a := range m.approvals {
		if status != "" && a.Status != status {
			continue
		}
		filtered = append(filtered, a)
	}
	slices.SortStableFunc(filtered, func(a, b types.ReleaseApproval) int {
		if a.CreatedAt.Equal(b.CreatedAt) {
			return 0
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		return -1
	})
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) Approve(_ context.Context, approvalID, reviewerID string) (*types.ReleaseApproval, error) {
	for i, a := range m.approvals {
		if a.ID == approvalID {
			m.approvals[i].Status = "approved"
			m.approvals[i].ReviewedBy = reviewerID
			m.approvals[i].ReviewedAt = time.Now().UTC()
			cp := m.approvals[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("approval not found")
}

func (m *MemoryRepository) Reject(_ context.Context, approvalID, reviewerID, reason string) (*types.ReleaseApproval, error) {
	for i, a := range m.approvals {
		if a.ID == approvalID {
			m.approvals[i].Status = "rejected"
			m.approvals[i].ReviewedBy = reviewerID
			m.approvals[i].ReviewedAt = time.Now().UTC()
			m.approvals[i].RejectionReason = reason
			cp := m.approvals[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("approval not found")
}

func (m *MemoryRepository) ListDrift(_ context.Context, orgID string, page, limit int) (types.Paginated[types.DriftItem], error) {
	_ = orgID
	filtered := make([]types.DriftItem, len(m.driftItems))
	copy(filtered, m.driftItems)
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) ListAuditEvents(_ context.Context, orgID string, page, limit int) (types.Paginated[types.AuditEvent], error) {
	filtered := make([]types.AuditEvent, 0)
	for _, e := range m.audit {
		if e.OrgID == orgID {
			filtered = append(filtered, e)
		}
	}
	return paginate(filtered, page, limit), nil
}

func (m *MemoryRepository) GetAuditEvent(_ context.Context, orgID, eventID string) (*types.AuditEvent, error) {
	for _, e := range m.audit {
		if e.OrgID == orgID && e.ID == eventID {
			cp := e
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("event not found")
}

func (m *MemoryRepository) GetAuditStats(_ context.Context, orgID, period string) (*types.AuditStats, error) {
	_ = period
	var total int
	for _, e := range m.audit {
		if e.OrgID == orgID {
			total++
		}
	}
	return &types.AuditStats{
		ActionStats:   []map[string]any{{"action": "update", "count": total}},
		ResourceStats: []map[string]any{{"resource_type": "Deployment", "count": total}},
		UserStats:     []map[string]any{{"username": "admin@kubeaudit.io", "count": total}},
		Timeline:      []map[string]any{{"hour": time.Now().Add(-1 * time.Hour).Format(time.RFC3339), "count": total}},
	}, nil
}

func (m *MemoryRepository) ListComplianceChecks(_ context.Context, orgID string) ([]types.ComplianceCheck, error) {
	_ = orgID
	out := make([]types.ComplianceCheck, len(m.compliance))
	copy(out, m.compliance)
	return out, nil
}

func (m *MemoryRepository) ListNotificationChannels(_ context.Context, orgID string) ([]types.NotificationChannel, error) {
	_ = orgID
	out := make([]types.NotificationChannel, len(m.channels))
	copy(out, m.channels)
	return out, nil
}

func (m *MemoryRepository) CreateNotificationChannel(_ context.Context, orgID, name, channelType string) (*types.NotificationChannel, error) {
	_ = orgID
	ch := types.NotificationChannel{ID: uuid.NewString(), Name: name, Type: channelType, Enabled: true, CreatedAt: time.Now().UTC()}
	m.channels = append(m.channels, ch)
	return &ch, nil
}

func (m *MemoryRepository) UpdateNotificationChannel(_ context.Context, orgID, channelID string, enabled *bool) (*types.NotificationChannel, error) {
	_ = orgID
	for i, ch := range m.channels {
		if ch.ID == channelID {
			if enabled != nil {
				m.channels[i].Enabled = *enabled
			}
			cp := m.channels[i]
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("channel not found")
}

func (m *MemoryRepository) DeleteNotificationChannel(_ context.Context, orgID, channelID string) error {
	_ = orgID
	for i, ch := range m.channels {
		if ch.ID == channelID {
			m.channels = append(m.channels[:i], m.channels[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("channel not found")
}

func (m *MemoryRepository) ListNotificationRules(_ context.Context, orgID string) ([]types.NotificationRule, error) {
	_ = orgID
	out := make([]types.NotificationRule, len(m.rules))
	copy(out, m.rules)
	return out, nil
}

func (m *MemoryRepository) CreateNotificationRule(_ context.Context, orgID, name string, events, channelIDs []string, filters map[string]any) (*types.NotificationRule, error) {
	_ = orgID
	r := types.NotificationRule{
		ID:         uuid.NewString(),
		Name:       name,
		Events:     events,
		ChannelIDs: channelIDs,
		Filters:    filters,
		Enabled:    true,
		CreatedAt:  time.Now().UTC(),
	}
	m.rules = append(m.rules, r)
	return &r, nil
}

func (m *MemoryRepository) ListReports(_ context.Context, orgID string) ([]types.Report, error) {
	_ = orgID
	out := make([]types.Report, len(m.reports))
	copy(out, m.reports)
	return out, nil
}

func (m *MemoryRepository) CreateReport(_ context.Context, orgID, userID, name, reportType, format string, filters map[string]any) (*types.Report, error) {
	_ = orgID
	_ = filters
	rep := types.Report{
		ID:        uuid.NewString(),
		Name:      name,
		Type:      reportType,
		Status:    "pending",
		Format:    format,
		CreatedBy: userID,
		CreatedAt: time.Now().UTC(),
	}
	m.reports = append([]types.Report{rep}, m.reports...)
	return &rep, nil
}

func (m *MemoryRepository) GetReport(_ context.Context, orgID, reportID string) (*types.Report, error) {
	_ = orgID
	for _, r := range m.reports {
		if r.ID == reportID {
			cp := r
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("report not found")
}

func (m *MemoryRepository) GetOrgSettings(_ context.Context, orgID string) (*types.OrgSettings, error) {
	if m.org.ID != orgID {
		return nil, fmt.Errorf("organization not found")
	}
	cp := m.org
	return &cp, nil
}

func (m *MemoryRepository) ListOrgUsers(_ context.Context, orgID string) ([]types.User, error) {
	out := make([]types.User, 0)
	for _, u := range m.users {
		if u.OrgID == orgID {
			cp := u
			cp.Password = ""
			out = append(out, cp)
		}
	}
	return out, nil
}

func (m *MemoryRepository) UpdateOrganization(_ context.Context, orgID, name string, settings map[string]any) (*types.OrgSettings, error) {
	if m.org.ID != orgID {
		return nil, fmt.Errorf("organization not found")
	}
	if name != "" {
		m.org.Name = name
	}
	if settings != nil {
		m.org.Settings = settings
	}
	cp := m.org
	return &cp, nil
}

func (m *MemoryRepository) UpdateUserRole(_ context.Context, orgID, userID, role string) (*types.User, error) {
	for i, u := range m.users {
		if u.OrgID == orgID && u.ID == userID {
			m.users[i].Role = role
			cp := m.users[i]
			cp.Password = ""
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// ── Helm repository management ────────────────────────────────────────────────

func (m *MemoryRepository) ListHelmRepositories(_ context.Context, orgID string) ([]types.HelmRepository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []types.HelmRepository
	for _, r := range m.helmRepos {
		if r.OrgID == orgID {
			cp := r
			cp.Credentials = nil
			out = append(out, cp)
		}
	}
	if out == nil {
		out = []types.HelmRepository{}
	}
	return out, nil
}

func (m *MemoryRepository) CreateHelmRepository(_ context.Context, r types.HelmRepository) (*types.HelmRepository, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	if r.Status == "" {
		r.Status = "pending"
	}
	m.helmRepos = append(m.helmRepos, r)
	cp := r
	cp.Credentials = nil
	return &cp, nil
}

func (m *MemoryRepository) UpdateHelmRepository(_ context.Context, orgID, repoID, url string, credentials map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.helmRepos {
		if r.OrgID == orgID && r.ID == repoID {
			m.helmRepos[i].URL = url
			m.helmRepos[i].Credentials = credentials
			m.helmRepos[i].Status = "pending"
			m.helmRepos[i].LastError = ""
			return nil
		}
	}
	return fmt.Errorf("repository not found")
}

func (m *MemoryRepository) DeleteHelmRepository(_ context.Context, orgID, repoID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.helmRepos {
		if r.OrgID == orgID && r.ID == repoID {
			m.helmRepos = slices.Delete(m.helmRepos, i, i+1)
			return nil
		}
	}
	return fmt.Errorf("repository not found")
}

func (m *MemoryRepository) UpdateHelmRepositoryStatus(_ context.Context, repoID, status, lastError string, lastSync time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.helmRepos {
		if r.ID == repoID {
			m.helmRepos[i].Status = status
			m.helmRepos[i].LastError = lastError
			m.helmRepos[i].LastSync = lastSync
			return nil
		}
	}
	return fmt.Errorf("repository not found")
}

func (m *MemoryRepository) GetHelmRepositoryCredentials(_ context.Context, orgID, repoID string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, r := range m.helmRepos {
		if r.OrgID == orgID && r.ID == repoID {
			return r.Credentials, nil
		}
	}
	return nil, fmt.Errorf("repository not found")
}

func (m *MemoryRepository) GetEnabledProviders(_ context.Context, orgID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp := make([]string, len(m.enabledProviders))
	copy(cp, m.enabledProviders)
	return cp, nil
}

func (m *MemoryRepository) SetEnabledProviders(_ context.Context, orgID string, providerIDs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabledProviders = make([]string, len(providerIDs))
	copy(m.enabledProviders, providerIDs)
	return nil
}

func paginate[T any](items []T, page, limit int) types.Paginated[T] {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	start := (page - 1) * limit
	if start > len(items) {
		start = len(items)
	}
	end := start + limit
	if end > len(items) {
		end = len(items)
	}

	res := types.Paginated[T]{Items: items[start:end]}
	res.Meta.Page = page
	res.Meta.Limit = limit
	res.Meta.Total = len(items)
	res.Meta.Pages = (len(items) + limit - 1) / limit
	if res.Meta.Pages == 0 {
		res.Meta.Pages = 1
	}
	return res
}
