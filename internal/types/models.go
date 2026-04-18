package types

import "time"

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Role     string `json:"role"`
	OrgID    string `json:"orgId"`
	Password string `json:"-"`
	Active   bool   `json:"-"`
}

type Cluster struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"orgId"`
	Name           string    `json:"name"`
	Provider       string    `json:"provider"`
	Environment    string    `json:"environment"`
	AuthType       string    `json:"authType"`
	Status         string    `json:"status"`
	ServerVersion  string    `json:"serverVersion,omitempty"`
	LastError      string    `json:"lastError,omitempty"`
	ReleaseCount   int       `json:"releaseCount"`
	NodeCount      int       `json:"nodeCount,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	KubeconfigRaw  string    `json:"-"`
}

type ClusterHealth struct {
	Nodes struct {
		Total    int `json:"total"`
		Ready    int `json:"ready"`
		NotReady int `json:"notReady"`
	} `json:"nodes"`
	Pods struct {
		Total     int `json:"total"`
		Running   int `json:"running"`
		Pending   int `json:"pending"`
		Failed    int `json:"failed"`
		Succeeded int `json:"succeeded"`
	} `json:"pods"`
	Timestamp string `json:"timestamp"`
	Error     string `json:"error,omitempty"`
}

type HelmRelease struct {
	ID           string    `json:"id"`
	ClusterID    string    `json:"clusterId"`
	ClusterName  string    `json:"clusterName"`
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	ChartName    string    `json:"chartName"`
	ChartVersion string    `json:"chartVersion"`
	AppVersion   string    `json:"appVersion,omitempty"`
	Status       string    `json:"status"`
	Revision     int       `json:"revision"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type ReleaseRevision struct {
	ID           string    `json:"id"`
	ReleaseID    string    `json:"releaseId"`
	Revision     int       `json:"revision"`
	ChartVersion string    `json:"chartVersion"`
	Status       string    `json:"status"`
	Description  string    `json:"description,omitempty"`
	Manifest     string    `json:"manifest,omitempty"`
	ValuesYAML   string    `json:"valuesYaml,omitempty"`
	DeployedAt   time.Time `json:"deployedAt"`
}

type DiffLine struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type DiffHunk struct {
	Header string     `json:"header"`
	Lines  []DiffLine `json:"lines"`
}

type DiffStats struct {
	Added     int `json:"added"`
	Removed   int `json:"removed"`
	Unchanged int `json:"unchanged"`
}

type DiffResult struct {
	Hunks []DiffHunk `json:"hunks"`
	Stats DiffStats  `json:"stats"`
	RevA  int        `json:"revA"`
	RevB  int        `json:"revB"`
}

type ReleaseApproval struct {
	ID              string    `json:"id"`
	ReleaseID       string    `json:"releaseId"`
	ReleaseName     string    `json:"releaseName"`
	Namespace       string    `json:"namespace"`
	ClusterName     string    `json:"clusterName"`
	RequestedBy     string    `json:"requestedBy"`
	ReviewedBy      string    `json:"reviewedBy,omitempty"`
	TargetVersion   string    `json:"targetVersion,omitempty"`
	Status          string    `json:"status"`
	RejectionReason string    `json:"rejectionReason,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	ReviewedAt      time.Time `json:"reviewedAt,omitempty"`
}

type DriftItem struct {
	ID          string    `json:"id"`
	ReleaseID   string    `json:"releaseId"`
	ReleaseName string    `json:"releaseName"`
	ClusterName string    `json:"clusterName"`
	Environment string    `json:"environment"`
	Status      string    `json:"status"`
	Diff        string    `json:"diff,omitempty"`
	DetectedAt  time.Time `json:"detectedAt"`
}

type AuditEvent struct {
	ID           string         `json:"id"`
	OrgID        string         `json:"orgId"`
	ClusterID    string         `json:"clusterId,omitempty"`
	ClusterName  string         `json:"clusterName,omitempty"`
	Username     string         `json:"username"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resourceType"`
	ResourceName string         `json:"resourceName,omitempty"`
	Namespace    string         `json:"namespace,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
	SourceIP     string         `json:"sourceIp,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
}

type AuditStats struct {
	ActionStats   []map[string]any `json:"actionStats"`
	ResourceStats []map[string]any `json:"resourceStats"`
	UserStats     []map[string]any `json:"userStats"`
	Timeline      []map[string]any `json:"timeline"`
}

type ComplianceCheck struct {
	ID        string         `json:"id"`
	Category  string         `json:"category"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Message   string         `json:"message,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
	CheckedAt time.Time      `json:"checkedAt"`
}

type NotificationChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
}

type NotificationRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Events     []string       `json:"events"`
	ChannelIDs []string       `json:"channelIds"`
	Filters    map[string]any `json:"filters"`
	Enabled    bool           `json:"enabled"`
	CreatedAt  time.Time      `json:"createdAt"`
}

type Report struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`
	FileSize    int       `json:"fileSize,omitempty"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	CompletedAt time.Time `json:"completedAt,omitempty"`
}

type OrgSettings struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Settings map[string]any `json:"settings,omitempty"`
}

type HelmRepository struct {
	ID          string            `json:"id"`
	OrgID       string            `json:"orgId"`
	Name        string            `json:"name"`
	URL         string            `json:"url"`
	ProviderID  string            `json:"providerId"`
	ProviderName string           `json:"providerName,omitempty"`
	Status      string            `json:"status"`
	LastError   string            `json:"lastError,omitempty"`
	LastSync    time.Time         `json:"lastSync,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	Credentials map[string]string `json:"-"`
}

type ChartInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	AppVersion  string   `json:"appVersion,omitempty"`
	Description string   `json:"description,omitempty"`
	Keywords    []string `json:"keywords,omitempty"`
	Icon        string   `json:"icon,omitempty"`
	RepoID      string   `json:"repoId,omitempty"`
	RepoName    string   `json:"repoName,omitempty"`
}

type Paginated[T any] struct {
	Items []T `json:"items"`
	Meta  struct {
		Page  int `json:"page"`
		Limit int `json:"limit"`
		Total int `json:"total"`
		Pages int `json:"pages"`
	} `json:"meta"`
}
