package providers

// FieldSpec describes a single credential or config field for a provider.
type FieldSpec struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
}

// ProviderSpec is the static metadata for a supported repository provider type.
type ProviderSpec struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Icon        string      `json:"icon"`
	Category    string      `json:"category"` // "self-hosted" | "cloud"
	IsOCI       bool        `json:"isOci"`
	Fields      []FieldSpec `json:"fields"`
}

// All is the built-in catalog of supported provider types.
var All = []ProviderSpec{
	{
		ID:          "harbor",
		Name:        "Harbor",
		Description: "VMware Harbor — open source cloud native registry with RBAC and replication.",
		Icon:        "⚓",
		Category:    "self-hosted",
		IsOCI:       false,
		Fields: []FieldSpec{
			{Key: "url", Label: "Registry URL", Placeholder: "https://harbor.example.com/chartrepo/myproject", Required: true},
			{Key: "username", Label: "Username", Placeholder: "admin", Required: true},
			{Key: "password", Label: "Password", Placeholder: "••••••••", Required: true, Secret: true},
		},
	},
	{
		ID:          "artifactory",
		Name:        "JFrog Artifactory",
		Description: "JFrog Artifactory — universal artifact manager supporting Helm chart repositories.",
		Icon:        "🐸",
		Category:    "self-hosted",
		IsOCI:       false,
		Fields: []FieldSpec{
			{Key: "url", Label: "Repository URL", Placeholder: "https://artifactory.example.com/artifactory/helm-local", Required: true},
			{Key: "username", Label: "Username", Placeholder: "admin", Required: true},
			{Key: "apiKey", Label: "API Key / Password", Placeholder: "AKCp…", Required: true, Secret: true},
		},
	},
	{
		ID:          "nexus",
		Name:        "Nexus Repository Manager",
		Description: "Sonatype Nexus — enterprise artifact repository with Helm hosted/proxy repos.",
		Icon:        "🗄",
		Category:    "self-hosted",
		IsOCI:       false,
		Fields: []FieldSpec{
			{Key: "url", Label: "Repository URL", Placeholder: "https://nexus.example.com/repository/helm-hosted", Required: true},
			{Key: "username", Label: "Username", Placeholder: "admin", Required: true},
			{Key: "password", Label: "Password", Placeholder: "••••••••", Required: true, Secret: true},
		},
	},
	{
		ID:          "ecr",
		Name:        "AWS ECR",
		Description: "Amazon Elastic Container Registry — OCI-compatible private registry with IAM auth.",
		Icon:        "☁",
		Category:    "cloud",
		IsOCI:       true,
		Fields: []FieldSpec{
			{Key: "accountId", Label: "AWS Account ID", Placeholder: "123456789012", Required: true},
			{Key: "region", Label: "AWS Region", Placeholder: "us-east-1", Required: true},
			{Key: "accessKeyId", Label: "Access Key ID", Placeholder: "AKIA…", Required: true},
			{Key: "secretAccessKey", Label: "Secret Access Key", Placeholder: "••••••••", Required: true, Secret: true},
		},
	},
}

// Find returns the ProviderSpec for the given ID.
func Find(id string) (ProviderSpec, bool) {
	for _, p := range All {
		if p.ID == id {
			return p, true
		}
	}
	return ProviderSpec{}, false
}
