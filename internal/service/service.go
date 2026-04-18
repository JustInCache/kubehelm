package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/ankushko/k8s-project-revamp/internal/cache"
	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/helm"
	helmProviders "github.com/ankushko/k8s-project-revamp/internal/helm/providers"
	k8sclient "github.com/ankushko/k8s-project-revamp/internal/k8s"
	"github.com/ankushko/k8s-project-revamp/internal/store"
	"github.com/ankushko/k8s-project-revamp/internal/types"
)

const (
	ttlReleaseSummaryMem   = 7 * time.Second
	ttlReleaseSummaryRedis = 10 * time.Second
	ttlWorkloadMem         = 12 * time.Second
	ttlWorkloadRedis       = 18 * time.Second
	ttlMetadataMem         = 25 * time.Second
	ttlMetadataRedis       = 45 * time.Second
)

type Service struct {
	repo      store.Repository
	redis     *cache.RedisCache
	mem       *cache.InMemoryCache
	coalescer *cache.Coalescer
	logger    *slog.Logger
	cfg       config.Config
	streams   *StreamHub
	helmSvc   *helm.Service
	k8sMgr    *k8sclient.Manager
}

func NewService(
	repo store.Repository,
	redis *cache.RedisCache,
	mem *cache.InMemoryCache,
	coalescer *cache.Coalescer,
	logger *slog.Logger,
	cfg config.Config,
	helmSvc *helm.Service,
	k8sMgr *k8sclient.Manager,
) *Service {
	s := &Service{
		repo:      repo,
		redis:     redis,
		mem:       mem,
		coalescer: coalescer,
		logger:    logger,
		cfg:       cfg,
		streams:   NewStreamHub(logger),
		helmSvc:   helmSvc,
		k8sMgr:    k8sMgr,
	}
	s.startStatusTicker()
	return s
}

func (s *Service) Ready(ctx context.Context) bool {
	return s.redis.Ping(ctx) == nil
}

// ─── Auth ───────────────────────────────────────────────────────────────────

func (s *Service) Login(ctx context.Context, email, password string) (map[string]any, error) {
	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}
	if strings.HasPrefix(user.Password, "$2") {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
			return nil, errors.New("invalid credentials")
		}
	} else if user.Password != password {
		return nil, errors.New("invalid credentials")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"orgId":  user.OrgID,
		"email":  user.Email,
		"role":   user.Role,
		"exp":    time.Now().Add(24 * time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}
	refresh := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID,
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	refreshSigned, err := refresh.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"token":        signed,
		"refreshToken": refreshSigned,
		"user": map[string]any{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
			"orgId": user.OrgID,
		},
	}, nil
}

func (s *Service) Me(ctx context.Context, userID string) (*types.User, error) {
	return s.repo.FindUserByID(ctx, userID)
}

// ─── Clusters ────────────────────────────────────────────────────────────────

func (s *Service) ListClusters(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.Cluster], error) {
	key := fmt.Sprintf("clusters:%s:%d:%d", orgID, page, limit)
	if v, ok := s.mem.Get(key); ok {
		return v.(types.Paginated[types.Cluster]), nil
	}
	var out types.Paginated[types.Cluster]
	if hit, _ := s.redis.Get(ctx, key, &out); hit {
		s.mem.Set(key, out, ttlReleaseSummaryMem)
		return out, nil
	}
	resAny, err, _ := s.coalescer.Do(ctx, key, func(ctx context.Context) (any, error) {
		res, err := s.repo.ListClusters(ctx, orgID, page, limit)
		if err != nil {
			return nil, err
		}
		_ = s.redis.Set(ctx, key, res, ttlReleaseSummaryRedis)
		s.mem.Set(key, res, ttlReleaseSummaryMem)
		return res, nil
	})
	if err != nil {
		return out, err
	}
	return resAny.(types.Paginated[types.Cluster]), nil
}

func (s *Service) GetCluster(ctx context.Context, orgID, clusterID string) (*types.Cluster, error) {
	return s.repo.GetCluster(ctx, orgID, clusterID)
}

// RegisterCluster tests connectivity then persists cluster with kubeconfig.
func (s *Service) RegisterCluster(ctx context.Context, orgID, name, provider, env, kubeconfigRaw string) (*types.Cluster, error) {
	serverVersion := ""
	status := "connected"
	lastErr := ""

	// Embed file-path cert references inline so the kubeconfig is self-contained
	// when written to a temp file for helm CLI invocations.
	if flat, err := k8sclient.FlattenKubeconfig(kubeconfigRaw); err != nil {
		s.logger.Warn("kubeconfig flatten failed", slog.String("err", err.Error()))
	} else {
		hasCertFile := strings.Contains(flat, "\n    certificate-authority:") || strings.Contains(flat, "\n  certificate-authority:")
		hasCertData := strings.Contains(flat, "certificate-authority-data:")
		s.logger.Info("kubeconfig flattened",
			slog.Int("original_len", len(kubeconfigRaw)),
			slog.Int("flat_len", len(flat)),
			slog.Bool("has_cert_file_ref", hasCertFile),
			slog.Bool("has_cert_data", hasCertData),
		)
		kubeconfigRaw = flat
	}

	if kubeconfigRaw != "" {
		ver, err := s.k8sMgr.TestConnection(ctx, kubeconfigRaw)
		if err != nil {
			status = "error"
			lastErr = err.Error()
		} else {
			serverVersion = ver
		}
	} else {
		status = "pending"
	}

	cluster := types.Cluster{
		OrgID:         orgID,
		Name:          name,
		Provider:      provider,
		Environment:   env,
		AuthType:      "kubeconfig",
		Status:        status,
		ServerVersion: serverVersion,
		LastError:     lastErr,
		KubeconfigRaw: kubeconfigRaw,
	}
	created, err := s.repo.CreateCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}
	s.invalidateClusterCache(orgID)
	return created, nil
}

// TestClusterConnection verifies a kubeconfig without storing it.
func (s *Service) TestClusterConnection(ctx context.Context, kubeconfigRaw string) (string, error) {
	return s.k8sMgr.TestConnection(ctx, kubeconfigRaw)
}

func (s *Service) DeleteCluster(ctx context.Context, orgID, clusterID, username string) error {
	if err := s.repo.DeleteCluster(ctx, orgID, clusterID); err != nil {
		return err
	}
	s.invalidateClusterCache(orgID)
	_ = s.logAudit(ctx, types.AuditEvent{
		OrgID:        orgID,
		ClusterID:    clusterID,
		Username:     username,
		Action:       "delete",
		ResourceType: "Cluster",
		ResourceName: clusterID,
	})
	return nil
}

func (s *Service) GetClusterHealth(ctx context.Context, orgID, clusterID string) (types.ClusterHealth, error) {
	key := fmt.Sprintf("health:%s:%s", orgID, clusterID)
	if v, ok := s.mem.Get(key); ok {
		return v.(types.ClusterHealth), nil
	}
	resAny, err, _ := s.coalescer.Do(ctx, key, func(ctx context.Context) (any, error) {
		kubeconfig, err := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
		if err != nil || kubeconfig == "" {
			// Return a placeholder health if no kubeconfig stored
			h := types.ClusterHealth{}
			h.Timestamp = time.Now().UTC().Format(time.RFC3339)
			h.Error = "no kubeconfig stored"
			return h, nil
		}
		health, err := s.k8sMgr.GetClusterHealth(ctx, kubeconfig)
		if err != nil {
			return types.ClusterHealth{Error: err.Error()}, nil
		}
		_ = s.redis.Set(ctx, key, health, ttlWorkloadRedis)
		s.mem.Set(key, health, ttlWorkloadMem)
		return health, nil
	})
	if err != nil {
		return types.ClusterHealth{}, err
	}
	return resAny.(types.ClusterHealth), nil
}

func (s *Service) GetClusterNodes(ctx context.Context, orgID, clusterID string) ([]map[string]any, error) {
	kubeconfig, err := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if err != nil || kubeconfig == "" {
		return []map[string]any{}, nil
	}
	return s.k8sMgr.GetNodes(ctx, kubeconfig)
}

func (s *Service) GetClusterNamespaces(ctx context.Context, orgID, clusterID string) ([]string, error) {
	kubeconfig, err := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if err != nil || kubeconfig == "" {
		return []string{"default", "kube-system"}, nil
	}
	return s.k8sMgr.GetNamespaces(ctx, kubeconfig)
}

// ─── Releases ────────────────────────────────────────────────────────────────

func (s *Service) ListReleases(ctx context.Context, orgID, namespace string, page, limit int, search, sortBy, sortOrder string) (types.Paginated[types.HelmRelease], error) {
	key := fmt.Sprintf("releases:%s:%s:%d:%d:%s:%s:%s", orgID, namespace, page, limit, search, sortBy, sortOrder)
	if v, ok := s.mem.Get(key); ok {
		return v.(types.Paginated[types.HelmRelease]), nil
	}

	// Try live Helm data first when helm CLI is available.
	if s.helmSvc.IsAvailable() {
		releases, err := s.fetchLiveReleases(ctx, orgID)
		if err == nil && len(releases) > 0 {
			filtered := filterReleases(releases, namespace, search)
			sortReleases(filtered, sortBy, sortOrder)
			result := paginateReleases(filtered, page, limit)
			s.mem.Set(key, result, ttlReleaseSummaryMem)
			return result, nil
		}
	}

	// Fall back to repository (memory or Postgres).
	var out types.Paginated[types.HelmRelease]
	hit, _ := s.redis.Get(ctx, key, &out)
	if hit {
		s.mem.Set(key, out, ttlReleaseSummaryMem)
		return out, nil
	}
	resAny, err, _ := s.coalescer.Do(ctx, key, func(ctx context.Context) (any, error) {
		res, err := s.repo.ListReleases(ctx, orgID, namespace, page, limit, search, sortBy, sortOrder)
		if err != nil {
			return nil, err
		}
		_ = s.redis.Set(ctx, key, res, ttlReleaseSummaryRedis)
		s.mem.Set(key, res, ttlReleaseSummaryMem)
		return res, nil
	})
	if err != nil {
		return out, err
	}
	return resAny.(types.Paginated[types.HelmRelease]), nil
}

func (s *Service) fetchLiveReleases(ctx context.Context, orgID string) ([]types.HelmRelease, error) {
	clusters, err := s.repo.ListClusters(ctx, orgID, 1, 1000)
	if err != nil {
		return nil, err
	}
	var all []types.HelmRelease
	for _, c := range clusters.Items {
		kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, c.ID)
		if kc == "" {
			s.logger.Warn("helm list skipped: no kubeconfig", slog.String("cluster", c.Name))
			continue
		}
		hasCertData := strings.Contains(kc, "certificate-authority-data:")
		hasCertFile := strings.Contains(kc, "\n    certificate-authority:")
		s.logger.Info("helm list attempt",
			slog.String("cluster", c.Name),
			slog.Bool("has_cert_data", hasCertData),
			slog.Bool("has_cert_file", hasCertFile),
			slog.Int("kubeconfig_len", len(kc)),
		)
		releases, err := s.helmSvc.ListReleases(ctx, kc, c.ID, c.Name)
		if err != nil {
			s.logger.Warn("helm list failed", slog.String("cluster", c.Name), slog.String("err", err.Error()))
			continue
		}
		all = append(all, releases...)
	}
	return all, nil
}

func filterReleases(releases []types.HelmRelease, namespace, search string) []types.HelmRelease {
	out := make([]types.HelmRelease, 0, len(releases))
	for _, r := range releases {
		if namespace != "" && namespace != "all" && r.Namespace != namespace {
			continue
		}
		if search != "" {
			s := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(r.Name), s) &&
				!strings.Contains(strings.ToLower(r.ChartName), s) {
				continue
			}
		}
		out = append(out, r)
	}
	return out
}

func sortReleases(releases []types.HelmRelease, sortBy, sortOrder string) {
	sign := 1
	if strings.EqualFold(sortOrder, "desc") {
		sign = -1
	}
	slices.SortStableFunc(releases, func(a, b types.HelmRelease) int {
		switch sortBy {
		case "name":
			return sign * strings.Compare(a.Name, b.Name)
		case "namespace":
			return sign * strings.Compare(a.Namespace, b.Namespace)
		case "status":
			return sign * strings.Compare(a.Status, b.Status)
		default:
			if a.UpdatedAt.Equal(b.UpdatedAt) {
				return 0
			}
			if a.UpdatedAt.Before(b.UpdatedAt) {
				return -1 * sign
			}
			return sign
		}
	})
}

func paginateReleases(items []types.HelmRelease, page, limit int) types.Paginated[types.HelmRelease] {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	res := types.Paginated[types.HelmRelease]{Items: items[start:end]}
	res.Meta.Page = page
	res.Meta.Limit = limit
	res.Meta.Total = total
	res.Meta.Pages = (total + limit - 1) / limit
	if res.Meta.Pages == 0 {
		res.Meta.Pages = 1
	}
	return res
}

// GetReleaseDetail returns history for a release, using live helm when available.
// releaseID format for live releases: "clusterID/namespace/releaseName"
func (s *Service) ListReleaseHistory(ctx context.Context, orgID, releaseID string, page, limit int) (types.Paginated[types.ReleaseRevision], error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if isLive && s.helmSvc.IsAvailable() {
		kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
		if kc != "" {
			revisions, err := s.helmSvc.History(ctx, kc, namespace, releaseName)
			if err == nil {
				return paginateRevisions(revisions, page, limit), nil
			}
		}
	}
	return s.repo.ListReleaseHistory(ctx, releaseID, page, limit)
}

func (s *Service) GetManifest(ctx context.Context, orgID, releaseID string, revision int) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if isLive && s.helmSvc.IsAvailable() {
		kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
		if kc != "" {
			return s.helmSvc.GetManifest(ctx, kc, namespace, releaseName, revision)
		}
	}
	return s.repo.GetReleaseManifest(ctx, releaseID, revision)
}

func (s *Service) Diff(ctx context.Context, orgID, releaseID string, revA, revB int) (types.DiffResult, error) {
	a, err := s.GetManifest(ctx, orgID, releaseID, revA)
	if err != nil {
		return types.DiffResult{}, err
	}
	b, err := s.GetManifest(ctx, orgID, releaseID, revB)
	if err != nil {
		return types.DiffResult{}, err
	}
	return lineDiff(a, b, revA, revB), nil
}

func (s *Service) DryRun(ctx context.Context, orgID, releaseID, chart, version string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "---\n# dry-run not available without helm CLI and kubeconfig", nil
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	return s.helmSvc.DryRun(ctx, kc, namespace, releaseName, chart, version)
}

func (s *Service) UpgradeRelease(ctx context.Context, orgID, releaseID, chart, version, username string, values map[string]string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "", errors.New("helm CLI not available or release ID not in live format")
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	output, err := s.helmSvc.Upgrade(ctx, kc, namespace, releaseName, chart, version, values)
	if err != nil {
		return output, err
	}
	s.invalidateReleaseCache(orgID)
	_ = s.logAudit(ctx, types.AuditEvent{
		OrgID:        orgID,
		ClusterID:    clusterID,
		Username:     username,
		Action:       "upgrade",
		ResourceType: "HelmRelease",
		ResourceName: releaseName,
		Namespace:    namespace,
		Details:      map[string]any{"chart": chart, "version": version},
	})
	return output, nil
}

func (s *Service) RollbackRelease(ctx context.Context, orgID, releaseID string, revision int, username string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "", errors.New("helm CLI not available or release ID not in live format")
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	output, err := s.helmSvc.Rollback(ctx, kc, namespace, releaseName, revision)
	if err != nil {
		return output, err
	}
	s.invalidateReleaseCache(orgID)
	_ = s.logAudit(ctx, types.AuditEvent{
		OrgID:        orgID,
		ClusterID:    clusterID,
		Username:     username,
		Action:       "rollback",
		ResourceType: "HelmRelease",
		ResourceName: releaseName,
		Namespace:    namespace,
		Details:      map[string]any{"revision": revision},
	})
	return output, nil
}

func (s *Service) GetReleaseValues(ctx context.Context, orgID, releaseID string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "", errors.New("helm CLI not available or release ID not in live format")
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	return s.helmSvc.GetValues(ctx, kc, namespace, releaseName)
}

func (s *Service) InstallRelease(ctx context.Context, orgID, clusterID, namespace, releaseName, chart, version, username string, values map[string]string) (string, error) {
	if !s.helmSvc.IsAvailable() {
		return "", errors.New("helm CLI not available")
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	output, err := s.helmSvc.Install(ctx, kc, namespace, releaseName, chart, version, values)
	if err != nil {
		return output, err
	}
	s.invalidateReleaseCache(orgID)
	_ = s.logAudit(ctx, types.AuditEvent{
		OrgID:        orgID,
		ClusterID:    clusterID,
		Username:     username,
		Action:       "install",
		ResourceType: "HelmRelease",
		ResourceName: releaseName,
		Namespace:    namespace,
		Details:      map[string]any{"chart": chart, "version": version},
	})
	return output, nil
}

func (s *Service) UninstallRelease(ctx context.Context, orgID, releaseID, username string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "", errors.New("helm CLI not available or release ID not in live format")
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	output, err := s.helmSvc.Uninstall(ctx, kc, namespace, releaseName)
	if err != nil {
		return output, err
	}
	s.invalidateReleaseCache(orgID)
	_ = s.logAudit(ctx, types.AuditEvent{
		OrgID:        orgID,
		ClusterID:    clusterID,
		Username:     username,
		Action:       "uninstall",
		ResourceType: "HelmRelease",
		ResourceName: releaseName,
		Namespace:    namespace,
	})
	return output, nil
}

func (s *Service) ListCharts(ctx context.Context, orgID, repoID string) ([]types.ChartInfo, error) {
	repos, err := s.repo.ListHelmRepositories(ctx, orgID)
	if err != nil {
		return nil, err
	}
	var target *types.HelmRepository
	for i := range repos {
		if repos[i].ID == repoID {
			target = &repos[i]
			break
		}
	}
	if target == nil {
		return nil, errors.New("repository not found")
	}
	if target.URL == "" {
		return nil, errors.New("OCI repositories do not support chart listing")
	}
	creds, _ := s.repo.GetHelmRepositoryCredentials(ctx, orgID, repoID)
	charts, err := s.helmSvc.ListCharts(ctx, target.URL, creds["username"], creds["password"])
	if err != nil {
		return nil, err
	}
	for i := range charts {
		charts[i].RepoID = repoID
		charts[i].RepoName = target.Name
	}
	return charts, nil
}

func (s *Service) TestRelease(ctx context.Context, orgID, releaseID string) (string, error) {
	clusterID, namespace, releaseName, isLive := parseReleaseID(releaseID)
	if !isLive || !s.helmSvc.IsAvailable() {
		return "helm test not available in this mode", nil
	}
	kc, _ := s.repo.GetClusterKubeconfig(ctx, orgID, clusterID)
	if kc == "" {
		return "", errors.New("no kubeconfig for cluster")
	}
	return s.helmSvc.RunTest(ctx, kc, namespace, releaseName)
}

// ─── Approvals ───────────────────────────────────────────────────────────────

func (s *Service) ListApprovals(ctx context.Context, orgID, status string, page, limit int) (types.Paginated[types.ReleaseApproval], error) {
	return s.repo.ListApprovals(ctx, orgID, status, page, limit)
}

func (s *Service) Approve(ctx context.Context, approvalID, reviewerID string) (*types.ReleaseApproval, error) {
	return s.repo.Approve(ctx, approvalID, reviewerID)
}

func (s *Service) Reject(ctx context.Context, approvalID, reviewerID, reason string) (*types.ReleaseApproval, error) {
	return s.repo.Reject(ctx, approvalID, reviewerID, reason)
}

// ─── Drift ───────────────────────────────────────────────────────────────────

func (s *Service) ListDrift(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.DriftItem], error) {
	return s.repo.ListDrift(ctx, orgID, page, limit)
}

// ─── Audit ───────────────────────────────────────────────────────────────────

func (s *Service) logAudit(ctx context.Context, e types.AuditEvent) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	return s.repo.CreateAuditEvent(ctx, e)
}

func (s *Service) ListAuditEvents(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.AuditEvent], error) {
	return s.repo.ListAuditEvents(ctx, orgID, page, limit)
}

func (s *Service) GetAuditEvent(ctx context.Context, orgID, eventID string) (*types.AuditEvent, error) {
	return s.repo.GetAuditEvent(ctx, orgID, eventID)
}

func (s *Service) GetAuditStats(ctx context.Context, orgID, period string) (*types.AuditStats, error) {
	return s.repo.GetAuditStats(ctx, orgID, period)
}

func (s *Service) ListCompliance(ctx context.Context, orgID string) ([]types.ComplianceCheck, error) {
	return s.repo.ListComplianceChecks(ctx, orgID)
}

// ─── Notifications ───────────────────────────────────────────────────────────

func (s *Service) ListChannels(ctx context.Context, orgID string) ([]types.NotificationChannel, error) {
	return s.repo.ListNotificationChannels(ctx, orgID)
}

func (s *Service) CreateChannel(ctx context.Context, orgID, name, channelType string) (*types.NotificationChannel, error) {
	return s.repo.CreateNotificationChannel(ctx, orgID, name, channelType)
}

func (s *Service) UpdateChannel(ctx context.Context, orgID, channelID string, enabled *bool) (*types.NotificationChannel, error) {
	return s.repo.UpdateNotificationChannel(ctx, orgID, channelID, enabled)
}

func (s *Service) DeleteChannel(ctx context.Context, orgID, channelID string) error {
	return s.repo.DeleteNotificationChannel(ctx, orgID, channelID)
}

func (s *Service) ListRules(ctx context.Context, orgID string) ([]types.NotificationRule, error) {
	return s.repo.ListNotificationRules(ctx, orgID)
}

func (s *Service) CreateRule(ctx context.Context, orgID, name string, events, channelIDs []string, filters map[string]any) (*types.NotificationRule, error) {
	return s.repo.CreateNotificationRule(ctx, orgID, name, events, channelIDs, filters)
}

// ─── Reports ─────────────────────────────────────────────────────────────────

func (s *Service) ListReports(ctx context.Context, orgID string) ([]types.Report, error) {
	return s.repo.ListReports(ctx, orgID)
}

func (s *Service) CreateReport(ctx context.Context, orgID, userID, name, reportType, format string, filters map[string]any) (*types.Report, error) {
	rep, err := s.repo.CreateReport(ctx, orgID, userID, name, reportType, format, filters)
	if err != nil {
		return nil, err
	}
	// Simulate async report completion after brief delay.
	go func() {
		time.Sleep(3 * time.Second)
		_ = s.repo.UpdateClusterStatus(context.Background(), rep.ID, "completed", "", "")
	}()
	return rep, nil
}

func (s *Service) GetReport(ctx context.Context, orgID, reportID string) (*types.Report, error) {
	return s.repo.GetReport(ctx, orgID, reportID)
}

// ─── Settings ────────────────────────────────────────────────────────────────

func (s *Service) GetSettings(ctx context.Context, orgID string) (*types.OrgSettings, []types.User, error) {
	org, err := s.repo.GetOrgSettings(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	users, err := s.repo.ListOrgUsers(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	return org, users, nil
}

func (s *Service) UpdateOrganization(ctx context.Context, orgID, name string, settings map[string]any) (*types.OrgSettings, error) {
	return s.repo.UpdateOrganization(ctx, orgID, name, settings)
}

func (s *Service) UpdateUserRole(ctx context.Context, orgID, userID, role string) (*types.User, error) {
	return s.repo.UpdateUserRole(ctx, orgID, userID, role)
}

// ─── SSE / Streams ───────────────────────────────────────────────────────────

func (s *Service) StreamHub() *StreamHub { return s.streams }

func (s *Service) startStatusTicker() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for ts := range t.C {
			s.streams.Broadcast(map[string]any{
				"type":      "status.tick",
				"timestamp": ts.UTC().Format(time.RFC3339),
				"health":    98 + rand.IntN(2),
			})
		}
	}()
}

// ─── Cache invalidation ───────────────────────────────────────────────────────

func (s *Service) invalidateClusterCache(orgID string) {
	s.mem.Delete("clusters:" + orgID + ":1:20")
}

func (s *Service) invalidateReleaseCache(orgID string) {
	// Broad invalidation — mem cache will expire naturally.
	_ = orgID
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// parseReleaseID checks if a release ID is a live "clusterID/namespace/name" triple.
func parseReleaseID(releaseID string) (clusterID, namespace, releaseName string, isLive bool) {
	parts := strings.SplitN(releaseID, "/", 3)
	if len(parts) == 3 {
		return parts[0], parts[1], parts[2], true
	}
	return "", "", "", false
}

func paginateRevisions(items []types.ReleaseRevision, page, limit int) types.Paginated[types.ReleaseRevision] {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	res := types.Paginated[types.ReleaseRevision]{Items: items[start:end]}
	res.Meta.Page = page
	res.Meta.Limit = limit
	res.Meta.Total = total
	res.Meta.Pages = (total + limit - 1) / limit
	if res.Meta.Pages == 0 {
		res.Meta.Pages = 1
	}
	return res
}

func lineDiff(a, b string, revA, revB int) types.DiffResult {
	la := strings.Split(a, "\n")
	lb := strings.Split(b, "\n")
	result := types.DiffResult{
		Hunks: []types.DiffHunk{{Header: "@@ manifest @@"}},
		RevA:  revA,
		RevB:  revB,
	}
	max := len(la)
	if len(lb) > max {
		max = len(lb)
	}
	for i := 0; i < max; i++ {
		var va, vb string
		if i < len(la) {
			va = la[i]
		}
		if i < len(lb) {
			vb = lb[i]
		}
		switch {
		case va == vb:
			if va != "" {
				result.Hunks[0].Lines = append(result.Hunks[0].Lines, types.DiffLine{Type: "unchanged", Content: va})
				result.Stats.Unchanged++
			}
		case va == "":
			result.Hunks[0].Lines = append(result.Hunks[0].Lines, types.DiffLine{Type: "added", Content: vb})
			result.Stats.Added++
		case vb == "":
			result.Hunks[0].Lines = append(result.Hunks[0].Lines, types.DiffLine{Type: "removed", Content: va})
			result.Stats.Removed++
		default:
			result.Hunks[0].Lines = append(result.Hunks[0].Lines, types.DiffLine{Type: "removed", Content: va})
			result.Hunks[0].Lines = append(result.Hunks[0].Lines, types.DiffLine{Type: "added", Content: vb})
			result.Stats.Removed++
			result.Stats.Added++
		}
	}
	return result
}

// ── Helm repository & provider management ─────────────────────────────────────

func (s *Service) ListHelmRepositories(ctx context.Context, orgID string) ([]types.HelmRepository, error) {
	repos, err := s.repo.ListHelmRepositories(ctx, orgID)
	if err != nil {
		return nil, err
	}
	// Enrich with provider name for display
	for i, r := range repos {
		if spec, ok := helmProviders.Find(r.ProviderID); ok {
			repos[i].ProviderName = spec.Name
		}
	}
	return repos, nil
}

func (s *Service) AddHelmRepository(ctx context.Context, orgID, name, url, providerID string, creds map[string]string) (*types.HelmRepository, error) {
	spec, ok := helmProviders.Find(providerID)
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerID)
	}

	r := types.HelmRepository{
		ID:          uuid.NewString(),
		OrgID:       orgID,
		Name:        name,
		URL:         url,
		ProviderID:  providerID,
		ProviderName: spec.Name,
		Credentials: creds,
		Status:      "pending",
	}

	created, err := s.repo.CreateHelmRepository(ctx, r)
	if err != nil {
		return nil, err
	}

	// Attempt the helm add immediately; update status based on outcome
	addErr := s.helmSvc.RepoAdd(ctx, name, url, providerID, creds)
	status, lastErr := "ok", ""
	if addErr != nil {
		status = "error"
		lastErr = addErr.Error()
		s.logger.Warn("helm repo add failed", slog.String("repo", name), slog.String("err", lastErr))
	}
	_ = s.repo.UpdateHelmRepositoryStatus(ctx, created.ID, status, lastErr, time.Now().UTC())
	created.Status = status
	created.LastError = lastErr
	created.ProviderName = spec.Name
	s.logAudit(ctx, types.AuditEvent{OrgID: orgID, Action: "create", ResourceType: "HelmRepository", ResourceName: name})
	return created, nil
}

func (s *Service) UpdateHelmRepository(ctx context.Context, orgID, repoID, url string, newCreds map[string]string) (*types.HelmRepository, error) {
	// Get existing credentials; merge — only overwrite keys that are non-empty in newCreds
	existingCreds, _ := s.repo.GetHelmRepositoryCredentials(ctx, orgID, repoID)
	if existingCreds == nil {
		existingCreds = map[string]string{}
	}
	for k, v := range newCreds {
		if v != "" {
			existingCreds[k] = v
		}
	}

	if err := s.repo.UpdateHelmRepository(ctx, orgID, repoID, url, existingCreds); err != nil {
		return nil, err
	}

	// Look up the repo to get its name and providerID for re-registering with helm
	repos, _ := s.repo.ListHelmRepositories(ctx, orgID)
	var found *types.HelmRepository
	for i := range repos {
		if repos[i].ID == repoID {
			found = &repos[i]
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("repository not found")
	}
	found.Credentials = existingCreds

	// Re-run helm repo add with updated URL/credentials
	addErr := s.helmSvc.RepoAdd(ctx, found.Name, url, found.ProviderID, existingCreds)
	status, lastErr := "ok", ""
	if addErr != nil {
		status = "error"
		lastErr = addErr.Error()
	}
	_ = s.repo.UpdateHelmRepositoryStatus(ctx, repoID, status, lastErr, time.Now().UTC())
	found.Status = status
	found.LastError = lastErr

	s.logAudit(ctx, types.AuditEvent{OrgID: orgID, Action: "update", ResourceType: "HelmRepository", ResourceName: found.Name})
	return found, nil
}

func (s *Service) RemoveHelmRepository(ctx context.Context, orgID, repoID string) error {
	creds, err := s.repo.GetHelmRepositoryCredentials(ctx, orgID, repoID)
	if err != nil {
		return err
	}
	repos, _ := s.repo.ListHelmRepositories(ctx, orgID)
	var repoName string
	for _, r := range repos {
		if r.ID == repoID {
			repoName = r.Name
			break
		}
	}
	if repoName != "" {
		_ = s.helmSvc.RepoRemove(ctx, repoName)
		_ = creds // consumed above if needed
	}
	if err := s.repo.DeleteHelmRepository(ctx, orgID, repoID); err != nil {
		return err
	}
	s.logAudit(ctx, types.AuditEvent{OrgID: orgID, Action: "delete", ResourceType: "HelmRepository", ResourceName: repoName})
	return nil
}

func (s *Service) RefreshHelmRepository(ctx context.Context, orgID, repoID string) error {
	creds, err := s.repo.GetHelmRepositoryCredentials(ctx, orgID, repoID)
	if err != nil {
		return err
	}
	repos, _ := s.repo.ListHelmRepositories(ctx, orgID)
	var repoName, providerID, url string
	for _, r := range repos {
		if r.ID == repoID {
			repoName, providerID, url = r.Name, r.ProviderID, r.URL
			break
		}
	}
	if repoName == "" {
		return fmt.Errorf("repository not found")
	}
	spec, _ := helmProviders.Find(providerID)
	var updateErr error
	if spec.IsOCI {
		// Re-login for ECR to refresh the short-lived token
		updateErr = s.helmSvc.RepoAdd(ctx, repoName, url, providerID, creds)
	} else {
		updateErr = s.helmSvc.RepoUpdate(ctx, repoName)
	}
	status, lastErr := "ok", ""
	if updateErr != nil {
		status, lastErr = "error", updateErr.Error()
	}
	return s.repo.UpdateHelmRepositoryStatus(ctx, repoID, status, lastErr, time.Now().UTC())
}

func (s *Service) TestHelmRepository(ctx context.Context, orgID, repoID string) error {
	creds, err := s.repo.GetHelmRepositoryCredentials(ctx, orgID, repoID)
	if err != nil {
		return err
	}
	repos, _ := s.repo.ListHelmRepositories(ctx, orgID)
	for _, r := range repos {
		if r.ID == repoID {
			return s.helmSvc.RepoTest(ctx, r.Name, r.URL, r.ProviderID, creds)
		}
	}
	return fmt.Errorf("repository not found")
}

func (s *Service) GetEnabledProviders(ctx context.Context, orgID string) ([]string, error) {
	return s.repo.GetEnabledProviders(ctx, orgID)
}

func (s *Service) SetEnabledProviders(ctx context.Context, orgID string, providerIDs []string) error {
	return s.repo.SetEnabledProviders(ctx, orgID, providerIDs)
}
