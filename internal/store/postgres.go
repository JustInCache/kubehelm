package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/types"
)

type PostgresRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewPostgresRepository(ctx context.Context, cfg config.Config, logger *slog.Logger) (*PostgresRepository, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode,
	)
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &PostgresRepository{pool: pool, logger: logger}, nil
}

func (p *PostgresRepository) Close() {
	p.pool.Close()
}

func (p *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (*types.User, error) {
	q := `SELECT id, email, name, role, org_id, is_active, COALESCE(password_hash,'') FROM users WHERE email = $1 LIMIT 1`
	var u types.User
	err := p.pool.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active, &u.Password)
	if err != nil {
		return nil, nil
	}
	return &u, nil
}

func (p *PostgresRepository) FindUserByID(ctx context.Context, id string) (*types.User, error) {
	q := `SELECT id, email, name, role, org_id, is_active FROM users WHERE id = $1 LIMIT 1`
	var u types.User
	err := p.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active)
	if err != nil {
		return nil, nil
	}
	return &u, nil
}

func (p *PostgresRepository) ListClusters(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.Cluster], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit
	countQ := `SELECT COUNT(*) FROM clusters WHERE org_id = $1`
	var total int
	_ = p.pool.QueryRow(ctx, countQ, orgID).Scan(&total)

	q := `
SELECT c.id, c.org_id, c.name, c.provider, c.environment, c.auth_type, c.status,
       COALESCE(c.server_version, ''), COALESCE(c.last_error, ''), c.created_at,
       (SELECT COUNT(*) FROM helm_releases hr WHERE hr.cluster_id = c.id)::int AS release_count
FROM clusters c
WHERE c.org_id = $1
ORDER BY c.environment, c.name
LIMIT $2 OFFSET $3`
	rows, err := p.pool.Query(ctx, q, orgID, limit, offset)
	if err != nil {
		return types.Paginated[types.Cluster]{}, err
	}
	defer rows.Close()
	items := make([]types.Cluster, 0)
	for rows.Next() {
		var c types.Cluster
		if err := rows.Scan(
			&c.ID, &c.OrgID, &c.Name, &c.Provider, &c.Environment, &c.AuthType, &c.Status,
			&c.ServerVersion, &c.LastError, &c.CreatedAt, &c.ReleaseCount,
		); err != nil {
			return types.Paginated[types.Cluster]{}, err
		}
		items = append(items, c)
	}
	return buildPaginated(items, page, limit, total), nil
}

func (p *PostgresRepository) GetCluster(ctx context.Context, orgID, clusterID string) (*types.Cluster, error) {
	q := `
SELECT c.id, c.org_id, c.name, c.provider, c.environment, c.auth_type, c.status,
       COALESCE(c.server_version, ''), COALESCE(c.last_error, ''), c.created_at,
       (SELECT COUNT(*) FROM helm_releases hr WHERE hr.cluster_id = c.id)::int AS release_count
FROM clusters c
WHERE c.id = $1 AND c.org_id = $2`
	var c types.Cluster
	err := p.pool.QueryRow(ctx, q, clusterID, orgID).Scan(
		&c.ID, &c.OrgID, &c.Name, &c.Provider, &c.Environment, &c.AuthType, &c.Status,
		&c.ServerVersion, &c.LastError, &c.CreatedAt, &c.ReleaseCount,
	)
	if err != nil {
		return nil, nil
	}
	return &c, nil
}

func (p *PostgresRepository) GetClusterKubeconfig(ctx context.Context, orgID, clusterID string) (string, error) {
	var raw string
	err := p.pool.QueryRow(ctx,
		`SELECT COALESCE(kubeconfig_raw,'') FROM clusters WHERE id=$1 AND org_id=$2`,
		clusterID, orgID).Scan(&raw)
	if err != nil {
		return "", nil
	}
	return raw, nil
}

func (p *PostgresRepository) CreateCluster(ctx context.Context, c types.Cluster) (*types.Cluster, error) {
	q := `INSERT INTO clusters (id, org_id, name, provider, environment, auth_type, status, server_version, last_error, kubeconfig_raw, created_at)
		  VALUES (gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		  RETURNING id, org_id, name, provider, environment, auth_type, status, COALESCE(server_version,''), COALESCE(last_error,''), created_at`
	var out types.Cluster
	err := p.pool.QueryRow(ctx, q,
		c.OrgID, c.Name, c.Provider, c.Environment, c.AuthType,
		c.Status, nullableString(c.ServerVersion), nullableString(c.LastError), nullableString(c.KubeconfigRaw),
	).Scan(&out.ID, &out.OrgID, &out.Name, &out.Provider, &out.Environment, &out.AuthType,
		&out.Status, &out.ServerVersion, &out.LastError, &out.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (p *PostgresRepository) UpdateClusterStatus(ctx context.Context, clusterID, status, serverVersion, lastError string) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE clusters SET status=$1, server_version=COALESCE($2,server_version), last_error=$3 WHERE id=$4`,
		status, nullableString(serverVersion), lastError, clusterID)
	return err
}

func (p *PostgresRepository) DeleteCluster(ctx context.Context, orgID, clusterID string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM clusters WHERE id=$1 AND org_id=$2`, clusterID, orgID)
	return err
}

func (p *PostgresRepository) CreateAuditEvent(ctx context.Context, event types.AuditEvent) error {
	det, _ := json.Marshal(event.Details)
	_, err := p.pool.Exec(ctx,
		`INSERT INTO audit_events (id, org_id, cluster_id, username, action, resource_type, resource_name, namespace, details, source_ip, created_at)
		 VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, NOW())`,
		event.OrgID, nullableString(event.ClusterID), event.Username,
		event.Action, event.ResourceType, event.ResourceName,
		event.Namespace, string(det), event.SourceIP)
	return err
}

func (p *PostgresRepository) ListReleases(ctx context.Context, orgID, namespace string, page, limit int, search, sortBy, sortOrder string) (types.Paginated[types.HelmRelease], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	sortColumn := mapSortRelease(sortBy)
	order := "DESC"
	if strings.EqualFold(sortOrder, "asc") {
		order = "ASC"
	}

	baseWhere := ` WHERE c.org_id = $1 `
	params := []any{orgID}
	if namespace != "" && namespace != "all" {
		params = append(params, namespace)
		baseWhere += fmt.Sprintf(" AND hr.namespace = $%d ", len(params))
	}
	if search != "" {
		params = append(params, "%"+search+"%")
		baseWhere += fmt.Sprintf(" AND (hr.name ILIKE $%d OR hr.chart_name ILIKE $%d) ", len(params), len(params))
	}

	countQ := `SELECT COUNT(*) FROM helm_releases hr JOIN clusters c ON c.id = hr.cluster_id` + baseWhere
	var total int
	_ = p.pool.QueryRow(ctx, countQ, params...).Scan(&total)

	params = append(params, limit, offset)
	q := fmt.Sprintf(`
SELECT hr.id, hr.cluster_id, c.name AS cluster_name, hr.name, hr.namespace,
       COALESCE(hr.chart_name, ''), COALESCE(hr.chart_version, ''), COALESCE(hr.status, ''), COALESCE(hr.revision, 0), hr.created_at
FROM helm_releases hr
JOIN clusters c ON c.id = hr.cluster_id
%s
ORDER BY %s %s
LIMIT $%d OFFSET $%d
`, baseWhere, sortColumn, order, len(params)-1, len(params))

	rows, err := p.pool.Query(ctx, q, params...)
	if err != nil {
		return types.Paginated[types.HelmRelease]{}, err
	}
	defer rows.Close()
	items := make([]types.HelmRelease, 0)
	for rows.Next() {
		var r types.HelmRelease
		if err := rows.Scan(&r.ID, &r.ClusterID, &r.ClusterName, &r.Name, &r.Namespace, &r.ChartName, &r.ChartVersion, &r.Status, &r.Revision, &r.CreatedAt); err != nil {
			return types.Paginated[types.HelmRelease]{}, err
		}
		items = append(items, r)
	}
	return buildPaginated(items, page, limit, total), nil
}

func (p *PostgresRepository) ListReleaseHistory(ctx context.Context, releaseID string, page, limit int) (types.Paginated[types.ReleaseRevision], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int
	_ = p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM helm_release_history WHERE release_id=$1`, releaseID).Scan(&total)
	rows, err := p.pool.Query(ctx, `
SELECT id, release_id, revision, COALESCE(chart_version, ''), COALESCE(status, ''), COALESCE(description, ''), COALESCE(manifest, ''), COALESCE(values_yaml, ''), COALESCE(deployed_at, NOW())
FROM helm_release_history
WHERE release_id=$1
ORDER BY revision DESC
LIMIT $2 OFFSET $3`, releaseID, limit, offset)
	if err != nil {
		return types.Paginated[types.ReleaseRevision]{}, err
	}
	defer rows.Close()
	items := make([]types.ReleaseRevision, 0)
	for rows.Next() {
		var h types.ReleaseRevision
		if err := rows.Scan(&h.ID, &h.ReleaseID, &h.Revision, &h.ChartVersion, &h.Status, &h.Description, &h.Manifest, &h.ValuesYAML, &h.DeployedAt); err != nil {
			return types.Paginated[types.ReleaseRevision]{}, err
		}
		items = append(items, h)
	}
	return buildPaginated(items, page, limit, total), nil
}

func (p *PostgresRepository) GetReleaseManifest(ctx context.Context, releaseID string, revision int) (string, error) {
	var manifest string
	err := p.pool.QueryRow(ctx, `SELECT COALESCE(manifest,'') FROM helm_release_history WHERE release_id=$1 AND revision=$2`, releaseID, revision).Scan(&manifest)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func (p *PostgresRepository) ListApprovals(ctx context.Context, orgID, status string, page, limit int) (types.Paginated[types.ReleaseApproval], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 100
	}
	offset := (page - 1) * limit
	params := []any{orgID}
	where := ` WHERE c.org_id=$1 `
	if status != "" {
		params = append(params, status)
		where += fmt.Sprintf(" AND ra.status=$%d ", len(params))
	}
	countQ := `SELECT COUNT(*) FROM release_approvals ra JOIN helm_releases hr ON hr.id=ra.release_id JOIN clusters c ON c.id=hr.cluster_id` + where
	var total int
	_ = p.pool.QueryRow(ctx, countQ, params...).Scan(&total)

	params = append(params, limit, offset)
	q := fmt.Sprintf(`
SELECT ra.id, ra.release_id, hr.name AS release_name, hr.namespace, c.name AS cluster_name,
       ra.requested_by, COALESCE(ra.reviewed_by::text,''), COALESCE(ra.target_version,''), ra.status,
       COALESCE(ra.rejection_reason,''), ra.created_at, COALESCE(ra.reviewed_at, ra.created_at)
FROM release_approvals ra
JOIN helm_releases hr ON hr.id=ra.release_id
JOIN clusters c ON c.id=hr.cluster_id
%s
ORDER BY ra.created_at DESC
LIMIT $%d OFFSET $%d`, where, len(params)-1, len(params))
	rows, err := p.pool.Query(ctx, q, params...)
	if err != nil {
		return types.Paginated[types.ReleaseApproval]{}, err
	}
	defer rows.Close()
	items := make([]types.ReleaseApproval, 0)
	for rows.Next() {
		var a types.ReleaseApproval
		if err := rows.Scan(&a.ID, &a.ReleaseID, &a.ReleaseName, &a.Namespace, &a.ClusterName, &a.RequestedBy, &a.ReviewedBy, &a.TargetVersion, &a.Status, &a.RejectionReason, &a.CreatedAt, &a.ReviewedAt); err != nil {
			return types.Paginated[types.ReleaseApproval]{}, err
		}
		items = append(items, a)
	}
	return buildPaginated(items, page, limit, total), nil
}

func (p *PostgresRepository) Approve(ctx context.Context, approvalID, reviewerID string) (*types.ReleaseApproval, error) {
	q := `
UPDATE release_approvals
SET status='approved', reviewed_by=$1, reviewed_at=NOW()
WHERE id=$2
RETURNING id, release_id, COALESCE(target_version,''), status, created_at, COALESCE(reviewed_at,created_at), COALESCE(rejection_reason,'')`
	var a types.ReleaseApproval
	a.ReviewedBy = reviewerID
	err := p.pool.QueryRow(ctx, q, reviewerID, approvalID).Scan(&a.ID, &a.ReleaseID, &a.TargetVersion, &a.Status, &a.CreatedAt, &a.ReviewedAt, &a.RejectionReason)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (p *PostgresRepository) Reject(ctx context.Context, approvalID, reviewerID, reason string) (*types.ReleaseApproval, error) {
	q := `
UPDATE release_approvals
SET status='rejected', reviewed_by=$1, reviewed_at=NOW(), rejection_reason=$2
WHERE id=$3
RETURNING id, release_id, COALESCE(target_version,''), status, created_at, COALESCE(reviewed_at,created_at), COALESCE(rejection_reason,'')`
	var a types.ReleaseApproval
	a.ReviewedBy = reviewerID
	err := p.pool.QueryRow(ctx, q, reviewerID, reason, approvalID).Scan(&a.ID, &a.ReleaseID, &a.TargetVersion, &a.Status, &a.CreatedAt, &a.ReviewedAt, &a.RejectionReason)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (p *PostgresRepository) ListDrift(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.DriftItem], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit
	countQ := `
SELECT COUNT(*)
FROM drift_detections dd
JOIN helm_releases hr ON hr.id=dd.release_id
JOIN clusters c ON c.id=hr.cluster_id
WHERE c.org_id=$1 AND dd.status='drifted'`
	var total int
	_ = p.pool.QueryRow(ctx, countQ, orgID).Scan(&total)

	rows, err := p.pool.Query(ctx, `
SELECT dd.id, dd.release_id, hr.name AS release_name, c.name AS cluster_name, c.environment, dd.status, COALESCE(dd.diff,''), dd.detected_at
FROM drift_detections dd
JOIN helm_releases hr ON hr.id=dd.release_id
JOIN clusters c ON c.id=hr.cluster_id
WHERE c.org_id=$1 AND dd.status='drifted'
ORDER BY dd.detected_at DESC
LIMIT $2 OFFSET $3`, orgID, limit, offset)
	if err != nil {
		return types.Paginated[types.DriftItem]{}, err
	}
	defer rows.Close()
	items := make([]types.DriftItem, 0)
	for rows.Next() {
		var d types.DriftItem
		if err := rows.Scan(&d.ID, &d.ReleaseID, &d.ReleaseName, &d.ClusterName, &d.Environment, &d.Status, &d.Diff, &d.DetectedAt); err != nil {
			return types.Paginated[types.DriftItem]{}, err
		}
		items = append(items, d)
	}
	return buildPaginated(items, page, limit, total), nil
}

func mapSortRelease(sortBy string) string {
	switch sortBy {
	case "name":
		return "hr.name"
	case "namespace":
		return "hr.namespace"
	case "status":
		return "hr.status"
	case "revision":
		return "hr.revision"
	default:
		return "hr.created_at"
	}
}

func buildPaginated[T any](items []T, page, limit, total int) types.Paginated[T] {
	res := types.Paginated[T]{Items: items}
	res.Meta.Page = page
	res.Meta.Limit = limit
	res.Meta.Total = total
	res.Meta.Pages = (total + limit - 1) / limit
	if res.Meta.Pages == 0 {
		res.Meta.Pages = 1
	}
	return res
}

func (p *PostgresRepository) ListAuditEvents(ctx context.Context, orgID string, page, limit int) (types.Paginated[types.AuditEvent], error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit
	var total int
	_ = p.pool.QueryRow(ctx, `SELECT COUNT(*) FROM audit_events WHERE org_id=$1`, orgID).Scan(&total)
	rows, err := p.pool.Query(ctx, `
SELECT ae.id, ae.org_id, COALESCE(ae.cluster_id::text,''), COALESCE(c.name,''), COALESCE(ae.username,''), COALESCE(ae.action,''), COALESCE(ae.resource_type,''), COALESCE(ae.resource_name,''), COALESCE(ae.namespace,''), COALESCE(ae.details::text,'{}'), COALESCE(ae.source_ip::text,''), ae.created_at
FROM audit_events ae
LEFT JOIN clusters c ON c.id = ae.cluster_id
WHERE ae.org_id=$1
ORDER BY ae.created_at DESC
LIMIT $2 OFFSET $3`, orgID, limit, offset)
	if err != nil {
		return types.Paginated[types.AuditEvent]{}, err
	}
	defer rows.Close()
	items := make([]types.AuditEvent, 0)
	for rows.Next() {
		var e types.AuditEvent
		var detailsStr string
		if err := rows.Scan(&e.ID, &e.OrgID, &e.ClusterID, &e.ClusterName, &e.Username, &e.Action, &e.ResourceType, &e.ResourceName, &e.Namespace, &detailsStr, &e.SourceIP, &e.CreatedAt); err != nil {
			return types.Paginated[types.AuditEvent]{}, err
		}
		_ = json.Unmarshal([]byte(detailsStr), &e.Details)
		items = append(items, e)
	}
	return buildPaginated(items, page, limit, total), nil
}

func (p *PostgresRepository) GetAuditEvent(ctx context.Context, orgID, eventID string) (*types.AuditEvent, error) {
	q := `
SELECT ae.id, ae.org_id, COALESCE(ae.cluster_id::text,''), COALESCE(c.name,''), COALESCE(ae.username,''), COALESCE(ae.action,''), COALESCE(ae.resource_type,''), COALESCE(ae.resource_name,''), COALESCE(ae.namespace,''), COALESCE(ae.details::text,'{}'), COALESCE(ae.source_ip::text,''), ae.created_at
FROM audit_events ae
LEFT JOIN clusters c ON c.id = ae.cluster_id
WHERE ae.org_id=$1 AND ae.id=$2`
	var e types.AuditEvent
	var detailsStr string
	if err := p.pool.QueryRow(ctx, q, orgID, eventID).Scan(&e.ID, &e.OrgID, &e.ClusterID, &e.ClusterName, &e.Username, &e.Action, &e.ResourceType, &e.ResourceName, &e.Namespace, &detailsStr, &e.SourceIP, &e.CreatedAt); err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(detailsStr), &e.Details)
	return &e, nil
}

func (p *PostgresRepository) GetAuditStats(ctx context.Context, orgID, period string) (*types.AuditStats, error) {
	if period == "" {
		period = "24 hours"
	}
	stats := &types.AuditStats{}
	actionRows, err := p.pool.Query(ctx, `SELECT action, COUNT(*)::int as count FROM audit_events WHERE org_id=$1 AND created_at >= NOW() - $2::interval GROUP BY action ORDER BY count DESC`, orgID, period)
	if err != nil {
		return nil, err
	}
	defer actionRows.Close()
	for actionRows.Next() {
		var action string
		var count int
		_ = actionRows.Scan(&action, &count)
		stats.ActionStats = append(stats.ActionStats, map[string]any{"action": action, "count": count})
	}
	stats.ResourceStats = []map[string]any{}
	stats.UserStats = []map[string]any{}
	stats.Timeline = []map[string]any{}
	return stats, nil
}

func (p *PostgresRepository) ListComplianceChecks(ctx context.Context, orgID string) ([]types.ComplianceCheck, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, category, name, status, COALESCE(message,''), COALESCE(details::text,'{}'), checked_at FROM compliance_checks WHERE org_id=$1 ORDER BY category,name`, orgID)
	if err != nil {
		return []types.ComplianceCheck{}, nil
	}
	defer rows.Close()
	items := make([]types.ComplianceCheck, 0)
	for rows.Next() {
		var c types.ComplianceCheck
		var details string
		if err := rows.Scan(&c.ID, &c.Category, &c.Name, &c.Status, &c.Message, &details, &c.CheckedAt); err == nil {
			_ = json.Unmarshal([]byte(details), &c.Details)
			items = append(items, c)
		}
	}
	return items, nil
}

func (p *PostgresRepository) ListNotificationChannels(ctx context.Context, orgID string) ([]types.NotificationChannel, error) {
	rows, err := p.pool.Query(ctx, `SELECT id, name, type, enabled, created_at FROM notification_channels WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return []types.NotificationChannel{}, nil
	}
	defer rows.Close()
	items := make([]types.NotificationChannel, 0)
	for rows.Next() {
		var c types.NotificationChannel
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Enabled, &c.CreatedAt); err == nil {
			items = append(items, c)
		}
	}
	return items, nil
}

func (p *PostgresRepository) CreateNotificationChannel(ctx context.Context, orgID, name, channelType string) (*types.NotificationChannel, error) {
	var c types.NotificationChannel
	err := p.pool.QueryRow(ctx, `INSERT INTO notification_channels (id, org_id, name, type, config_enc, enabled, created_at) VALUES (gen_random_uuid(),$1,$2,$3,'{}',true,NOW()) RETURNING id,name,type,enabled,created_at`, orgID, name, channelType).
		Scan(&c.ID, &c.Name, &c.Type, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *PostgresRepository) UpdateNotificationChannel(ctx context.Context, orgID, channelID string, enabled *bool) (*types.NotificationChannel, error) {
	var c types.NotificationChannel
	err := p.pool.QueryRow(ctx, `UPDATE notification_channels SET enabled=COALESCE($1, enabled) WHERE org_id=$2 AND id=$3 RETURNING id,name,type,enabled,created_at`, enabled, orgID, channelID).
		Scan(&c.ID, &c.Name, &c.Type, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (p *PostgresRepository) DeleteNotificationChannel(ctx context.Context, orgID, channelID string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM notification_channels WHERE org_id=$1 AND id=$2`, orgID, channelID)
	return err
}

func (p *PostgresRepository) ListNotificationRules(ctx context.Context, orgID string) ([]types.NotificationRule, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,name,events::text,channel_ids::text,filters::text,enabled,created_at FROM notification_rules WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return []types.NotificationRule{}, nil
	}
	defer rows.Close()
	items := make([]types.NotificationRule, 0)
	for rows.Next() {
		var r types.NotificationRule
		var eventsText, channelText, filtersText string
		if err := rows.Scan(&r.ID, &r.Name, &eventsText, &channelText, &filtersText, &r.Enabled, &r.CreatedAt); err == nil {
			_ = json.Unmarshal([]byte(eventsText), &r.Events)
			_ = json.Unmarshal([]byte(channelText), &r.ChannelIDs)
			_ = json.Unmarshal([]byte(filtersText), &r.Filters)
			items = append(items, r)
		}
	}
	return items, nil
}

func (p *PostgresRepository) CreateNotificationRule(ctx context.Context, orgID, name string, events, channelIDs []string, filters map[string]any) (*types.NotificationRule, error) {
	ev, _ := json.Marshal(events)
	ch, _ := json.Marshal(channelIDs)
	fb, _ := json.Marshal(filters)
	var r types.NotificationRule
	err := p.pool.QueryRow(ctx, `INSERT INTO notification_rules (id,org_id,name,events,channel_ids,filters,enabled,created_at) VALUES (gen_random_uuid(),$1,$2,$3::jsonb,$4::jsonb,$5::jsonb,true,NOW()) RETURNING id,name,enabled,created_at`, orgID, name, string(ev), string(ch), string(fb)).
		Scan(&r.ID, &r.Name, &r.Enabled, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	r.Events = events
	r.ChannelIDs = channelIDs
	r.Filters = filters
	return &r, nil
}

func (p *PostgresRepository) ListReports(ctx context.Context, orgID string) ([]types.Report, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,name,type,status,format,COALESCE(file_size,0),COALESCE(created_by::text,''),created_at,COALESCE(completed_at,created_at) FROM reports WHERE org_id=$1 ORDER BY created_at DESC LIMIT 100`, orgID)
	if err != nil {
		return []types.Report{}, nil
	}
	defer rows.Close()
	items := make([]types.Report, 0)
	for rows.Next() {
		var r types.Report
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Status, &r.Format, &r.FileSize, &r.CreatedBy, &r.CreatedAt, &r.CompletedAt); err == nil {
			items = append(items, r)
		}
	}
	return items, nil
}

func (p *PostgresRepository) CreateReport(ctx context.Context, orgID, userID, name, reportType, format string, filters map[string]any) (*types.Report, error) {
	fb, _ := json.Marshal(filters)
	var r types.Report
	err := p.pool.QueryRow(ctx, `INSERT INTO reports (id,org_id,name,type,format,status,filters,created_by,created_at) VALUES (gen_random_uuid(),$1,$2,$3,$4,'pending',$5::jsonb,$6,NOW()) RETURNING id,name,type,status,format,COALESCE(file_size,0),COALESCE(created_by::text,''),created_at,COALESCE(completed_at,created_at)`, orgID, name, reportType, format, string(fb), userID).
		Scan(&r.ID, &r.Name, &r.Type, &r.Status, &r.Format, &r.FileSize, &r.CreatedBy, &r.CreatedAt, &r.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (p *PostgresRepository) GetReport(ctx context.Context, orgID, reportID string) (*types.Report, error) {
	var r types.Report
	err := p.pool.QueryRow(ctx, `SELECT id,name,type,status,format,COALESCE(file_size,0),COALESCE(created_by::text,''),created_at,COALESCE(completed_at,created_at) FROM reports WHERE org_id=$1 AND id=$2`, orgID, reportID).
		Scan(&r.ID, &r.Name, &r.Type, &r.Status, &r.Format, &r.FileSize, &r.CreatedBy, &r.CreatedAt, &r.CompletedAt)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (p *PostgresRepository) GetOrgSettings(ctx context.Context, orgID string) (*types.OrgSettings, error) {
	var org types.OrgSettings
	var settingsStr string
	err := p.pool.QueryRow(ctx, `SELECT id,name,COALESCE(settings::text,'{}') FROM organizations WHERE id=$1`, orgID).Scan(&org.ID, &org.Name, &settingsStr)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(settingsStr), &org.Settings)
	return &org, nil
}

func (p *PostgresRepository) ListOrgUsers(ctx context.Context, orgID string) ([]types.User, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,email,name,role,org_id,is_active FROM users WHERE org_id=$1 ORDER BY name`, orgID)
	if err != nil {
		return []types.User{}, nil
	}
	defer rows.Close()
	items := make([]types.User, 0)
	for rows.Next() {
		var u types.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active); err == nil {
			items = append(items, u)
		}
	}
	return items, nil
}

func (p *PostgresRepository) UpdateOrganization(ctx context.Context, orgID, name string, settings map[string]any) (*types.OrgSettings, error) {
	sb, _ := json.Marshal(settings)
	var org types.OrgSettings
	var outSettings string
	err := p.pool.QueryRow(ctx, `UPDATE organizations SET name=COALESCE($1,name), settings=COALESCE($2::jsonb,settings) WHERE id=$3 RETURNING id,name,COALESCE(settings::text,'{}')`, nullableString(name), nullableJSON(string(sb), settings != nil), orgID).
		Scan(&org.ID, &org.Name, &outSettings)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal([]byte(outSettings), &org.Settings)
	return &org, nil
}

func (p *PostgresRepository) UpdateUserRole(ctx context.Context, orgID, userID, role string) (*types.User, error) {
	var u types.User
	err := p.pool.QueryRow(ctx, `UPDATE users SET role=$1 WHERE org_id=$2 AND id=$3 RETURNING id,email,name,role,org_id,is_active`, role, orgID, userID).
		Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.OrgID, &u.Active)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// ── Helm repository management ────────────────────────────────────────────────

func (p *PostgresRepository) ListHelmRepositories(ctx context.Context, orgID string) ([]types.HelmRepository, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, org_id, name, url, provider_id, status, last_error, last_sync, created_at
		 FROM helm_repositories WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []types.HelmRepository
	for rows.Next() {
		var r types.HelmRepository
		var lastErr *string
		var lastSync *time.Time
		if err := rows.Scan(&r.ID, &r.OrgID, &r.Name, &r.URL, &r.ProviderID,
			&r.Status, &lastErr, &lastSync, &r.CreatedAt); err != nil {
			return nil, err
		}
		if lastErr != nil {
			r.LastError = *lastErr
		}
		if lastSync != nil {
			r.LastSync = *lastSync
		}
		out = append(out, r)
	}
	if out == nil {
		out = []types.HelmRepository{}
	}
	return out, nil
}

func (p *PostgresRepository) CreateHelmRepository(ctx context.Context, r types.HelmRepository) (*types.HelmRepository, error) {
	credsJSON, _ := json.Marshal(r.Credentials)
	var created types.HelmRepository
	err := p.pool.QueryRow(ctx,
		`INSERT INTO helm_repositories (id, org_id, name, url, provider_id, credentials_json, status, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,'pending',NOW())
		 RETURNING id, org_id, name, url, provider_id, status, created_at`,
		r.ID, r.OrgID, r.Name, r.URL, r.ProviderID, string(credsJSON)).
		Scan(&created.ID, &created.OrgID, &created.Name, &created.URL, &created.ProviderID, &created.Status, &created.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (p *PostgresRepository) UpdateHelmRepository(ctx context.Context, orgID, repoID, url string, credentials map[string]string) error {
	credsJSON, _ := json.Marshal(credentials)
	_, err := p.pool.Exec(ctx,
		`UPDATE helm_repositories SET url=$1, credentials_json=$2, status='pending', last_error=NULL WHERE org_id=$3 AND id=$4`,
		url, string(credsJSON), orgID, repoID)
	return err
}

func (p *PostgresRepository) DeleteHelmRepository(ctx context.Context, orgID, repoID string) error {
	_, err := p.pool.Exec(ctx, `DELETE FROM helm_repositories WHERE org_id=$1 AND id=$2`, orgID, repoID)
	return err
}

func (p *PostgresRepository) UpdateHelmRepositoryStatus(ctx context.Context, repoID, status, lastError string, lastSync time.Time) error {
	_, err := p.pool.Exec(ctx,
		`UPDATE helm_repositories SET status=$1, last_error=NULLIF($2,''), last_sync=NULLIF($3::timestamptz, '0001-01-01'::timestamptz) WHERE id=$4`,
		status, lastError, lastSync, repoID)
	return err
}

func (p *PostgresRepository) GetHelmRepositoryCredentials(ctx context.Context, orgID, repoID string) (map[string]string, error) {
	var raw string
	err := p.pool.QueryRow(ctx,
		`SELECT COALESCE(credentials_json,'{}') FROM helm_repositories WHERE org_id=$1 AND id=$2`,
		orgID, repoID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	var creds map[string]string
	_ = json.Unmarshal([]byte(raw), &creds)
	return creds, nil
}

func (p *PostgresRepository) GetEnabledProviders(ctx context.Context, orgID string) ([]string, error) {
	var raw string
	err := p.pool.QueryRow(ctx,
		`SELECT COALESCE(settings->>'enabledProviders','[]') FROM organizations WHERE id=$1`, orgID).Scan(&raw)
	if err != nil {
		return []string{}, nil
	}
	var ids []string
	_ = json.Unmarshal([]byte(raw), &ids)
	return ids, nil
}

func (p *PostgresRepository) SetEnabledProviders(ctx context.Context, orgID string, providerIDs []string) error {
	b, _ := json.Marshal(providerIDs)
	_, err := p.pool.Exec(ctx,
		`UPDATE organizations SET settings = COALESCE(settings,'{}') || jsonb_build_object('enabledProviders',$1::jsonb) WHERE id=$2`,
		string(b), orgID)
	return err
}

func nullableString(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullableJSON(v string, present bool) any {
	if !present {
		return nil
	}
	return v
}
