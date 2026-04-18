package store

import (
	"context"
	"time"

	"github.com/ankushko/k8s-project-revamp/internal/types"
)

type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (*types.User, error)
	FindUserByID(ctx context.Context, id string) (*types.User, error)

	ListClusters(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.Cluster], error)
	GetCluster(ctx context.Context, orgID, clusterID string) (*types.Cluster, error)
	GetClusterKubeconfig(ctx context.Context, orgID, clusterID string) (string, error)
	CreateCluster(ctx context.Context, c types.Cluster) (*types.Cluster, error)
	UpdateClusterStatus(ctx context.Context, clusterID, status, serverVersion, lastError string) error
	DeleteCluster(ctx context.Context, orgID, clusterID string) error

	ListReleases(ctx context.Context, orgID, namespace string, page, limit int, search, sortBy, sortOrder string) (types.Paginated[types.HelmRelease], error)
	ListReleaseHistory(ctx context.Context, releaseID string, page, limit int) (types.Paginated[types.ReleaseRevision], error)
	GetReleaseManifest(ctx context.Context, releaseID string, revision int) (string, error)

	ListApprovals(ctx context.Context, orgID, status string, page, limit int) (types.Paginated[types.ReleaseApproval], error)
	Approve(ctx context.Context, approvalID, reviewerID string) (*types.ReleaseApproval, error)
	Reject(ctx context.Context, approvalID, reviewerID, reason string) (*types.ReleaseApproval, error)

	ListDrift(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.DriftItem], error)

	CreateAuditEvent(ctx context.Context, event types.AuditEvent) error
	ListAuditEvents(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.AuditEvent], error)
	GetAuditEvent(ctx context.Context, orgID, eventID string) (*types.AuditEvent, error)
	GetAuditStats(ctx context.Context, orgID, period string) (*types.AuditStats, error)
	ListComplianceChecks(ctx context.Context, orgID string) ([]types.ComplianceCheck, error)

	ListNotificationChannels(ctx context.Context, orgID string) ([]types.NotificationChannel, error)
	CreateNotificationChannel(ctx context.Context, orgID, name, channelType string) (*types.NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, orgID, channelID string, enabled *bool) (*types.NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, orgID, channelID string) error

	ListNotificationRules(ctx context.Context, orgID string) ([]types.NotificationRule, error)
	CreateNotificationRule(ctx context.Context, orgID, name string, events, channelIDs []string, filters map[string]any) (*types.NotificationRule, error)

	ListReports(ctx context.Context, orgID string) ([]types.Report, error)
	CreateReport(ctx context.Context, orgID, userID, name, reportType, format string, filters map[string]any) (*types.Report, error)
	GetReport(ctx context.Context, orgID, reportID string) (*types.Report, error)

	GetOrgSettings(ctx context.Context, orgID string) (*types.OrgSettings, error)
	ListOrgUsers(ctx context.Context, orgID string) ([]types.User, error)
	UpdateOrganization(ctx context.Context, orgID, name string, settings map[string]any) (*types.OrgSettings, error)
	UpdateUserRole(ctx context.Context, orgID, userID, role string) (*types.User, error)

	// Helm repository management
	ListHelmRepositories(ctx context.Context, orgID string) ([]types.HelmRepository, error)
	CreateHelmRepository(ctx context.Context, r types.HelmRepository) (*types.HelmRepository, error)
	UpdateHelmRepository(ctx context.Context, orgID, repoID, url string, credentials map[string]string) error
	DeleteHelmRepository(ctx context.Context, orgID, repoID string) error
	UpdateHelmRepositoryStatus(ctx context.Context, repoID, status, lastError string, lastSync time.Time) error
	GetHelmRepositoryCredentials(ctx context.Context, orgID, repoID string) (map[string]string, error)

	// Provider plugin enable/disable per org
	GetEnabledProviders(ctx context.Context, orgID string) ([]string, error)
	SetEnabledProviders(ctx context.Context, orgID string, providerIDs []string) error
}
