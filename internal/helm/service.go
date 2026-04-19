package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	helmrelease "helm.sh/helm/v3/pkg/release"
	helmdriver "helm.sh/helm/v3/pkg/storage/driver"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ankushko/k8s-project-revamp/internal/helm/providers"
	"github.com/ankushko/k8s-project-revamp/internal/types"
)

// runHelm runs a helm command capturing stdout; stderr is captured separately
// and appended to the error message on failure so the real cause is visible in logs.
func runHelm(cmd *exec.Cmd) ([]byte, error) {
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, fmt.Errorf("%w", err)
		}
		return nil, fmt.Errorf("%w: %s", err, msg)
	}
	return out, nil
}

type releaseLive struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   string `json:"revision"` // helm returns revision as a string in JSON output
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"app_version"`
}

type historyItem struct {
	Revision    int    `json:"revision"`
	Updated     string `json:"updated"`
	Status      string `json:"status"`
	Chart       string `json:"chart"`
	AppVersion  string `json:"app_version"`
	Description string `json:"description"`
}

// Service wraps the helm CLI binary. All operations require helm to be installed.
type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) IsAvailable() bool {
	_, err := exec.LookPath("helm")
	return err == nil
}

func (s *Service) withKubeconfig(raw string, fn func(path string) error) error {
	f, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return fmt.Errorf("create temp kubeconfig: %w", err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(raw); err != nil {
		f.Close()
		return err
	}
	f.Close()
	// Debug: log whether the kubeconfig has embedded certs or file-path refs
	hasCertData := strings.Contains(raw, "certificate-authority-data:")
	hasCertFile := strings.Contains(raw, "certificate-authority:")
	_ = hasCertData
	_ = hasCertFile
	return fn(f.Name())
}

// ListReleases queries Kubernetes secrets directly (where Helm stores its state)
// using client-go. This uses the same auth path as every other Kubernetes call
// in this application and avoids the action.Configuration RESTClientGetter path
// which fails with anonymous auth in certain kubeconfig configurations.
func (s *Service) ListReleases(ctx context.Context, kubeconfigRaw, clusterID, clusterName string) ([]types.HelmRelease, error) {
	restCfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigRaw))
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	restCfg.Timeout = 15 * time.Second

	cs, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("build k8s client: %w", err)
	}

	// Use the Helm storage driver backed by our working kubernetes client.
	// Passing "" as the namespace lists secrets across all namespaces.
	d := helmdriver.NewSecrets(cs.CoreV1().Secrets(""))
	list, err := d.List(func(_ *helmrelease.Release) bool { return true })
	if err != nil {
		return nil, fmt.Errorf("helm list: %w", err)
	}

	releases := make([]types.HelmRelease, 0, len(list))
	for _, r := range list {
		releases = append(releases, helmReleaseToType(r, clusterID, clusterName))
	}
	return releases, nil
}

func helmReleaseToType(r *helmrelease.Release, clusterID, clusterName string) types.HelmRelease {
	chartName := ""
	chartVersion := ""
	appVersion := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		chartName = r.Chart.Metadata.Name
		chartVersion = r.Chart.Metadata.Version
		appVersion = r.Chart.Metadata.AppVersion
	}
	updated := r.Info.LastDeployed.Time
	return types.HelmRelease{
		ID:           clusterID + "/" + r.Namespace + "/" + r.Name,
		ClusterID:    clusterID,
		ClusterName:  clusterName,
		Name:         r.Name,
		Namespace:    r.Namespace,
		ChartName:    chartName,
		ChartVersion: chartVersion,
		AppVersion:   appVersion,
		Status:       string(r.Info.Status),
		Revision:     r.Version,
		UpdatedAt:    updated,
		CreatedAt:    updated,
	}
}

func (s *Service) History(ctx context.Context, kubeconfigRaw, namespace, releaseName string) ([]types.ReleaseRevision, error) {
	var revisions []types.ReleaseRevision
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		cmd := exec.CommandContext(ctx, "helm", "history", releaseName,
			"-n", namespace, "--kubeconfig", kp, "-o", "json")
		out, err := runHelm(cmd)
		if err != nil {
			return fmt.Errorf("helm history: %w", err)
		}
		var raw []historyItem
		if err := json.Unmarshal(out, &raw); err != nil {
			return fmt.Errorf("parse helm history: %w", err)
		}
		releaseID := namespace + "/" + releaseName
		for _, h := range raw {
			_, chartVersion := parseChart(h.Chart)
			revisions = append(revisions, types.ReleaseRevision{
				ID:           fmt.Sprintf("%s/%d", releaseID, h.Revision),
				ReleaseID:    releaseID,
				Revision:     h.Revision,
				ChartVersion: chartVersion,
				Status:       h.Status,
				Description:  h.Description,
				DeployedAt:   parseHelmTime(h.Updated),
			})
		}
		return nil
	})
	return revisions, err
}

func (s *Service) GetManifest(ctx context.Context, kubeconfigRaw, namespace, releaseName string, revision int) (string, error) {
	var manifest string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		args := []string{"get", "manifest", releaseName, "-n", namespace, "--kubeconfig", kp}
		if revision > 0 {
			args = append(args, "--revision", fmt.Sprintf("%d", revision))
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		out, err := runHelm(cmd)
		if err != nil {
			return fmt.Errorf("helm get manifest: %w", err)
		}
		manifest = string(out)
		return nil
	})
	return manifest, err
}

func (s *Service) GetValues(ctx context.Context, kubeconfigRaw, namespace, releaseName string) (string, error) {
	var values string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		cmd := exec.CommandContext(ctx, "helm", "get", "values", releaseName,
			"-n", namespace, "--kubeconfig", kp)
		out, err := runHelm(cmd)
		if err != nil {
			return fmt.Errorf("helm get values: %w", err)
		}
		values = string(out)
		return nil
	})
	return values, err
}

func (s *Service) DryRun(ctx context.Context, kubeconfigRaw, namespace, releaseName, chart, version string) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		args := []string{"upgrade", releaseName, chart, "-n", namespace,
			"--kubeconfig", kp, "--install", "--dry-run"}
		if version != "" {
			args = append(args, "--version", version)
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm dry-run: %s", output)
		}
		return nil
	})
	return output, err
}

func (s *Service) Upgrade(ctx context.Context, kubeconfigRaw, namespace, releaseName, chart, version string, setValues map[string]string) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		args := []string{"upgrade", releaseName, chart, "-n", namespace,
			"--kubeconfig", kp, "--install", "--atomic", "--timeout", "5m"}
		if version != "" {
			args = append(args, "--version", version)
		}
		for k, v := range setValues {
			args = append(args, "--set", k+"="+v)
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm upgrade failed: %s", output)
		}
		return nil
	})
	return output, err
}

func (s *Service) Rollback(ctx context.Context, kubeconfigRaw, namespace, releaseName string, revision int) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		args := []string{"rollback", releaseName, fmt.Sprintf("%d", revision),
			"-n", namespace, "--kubeconfig", kp, "--timeout", "5m"}
		cmd := exec.CommandContext(ctx, "helm", args...)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm rollback failed: %s", output)
		}
		return nil
	})
	return output, err
}

func (s *Service) RunTest(ctx context.Context, kubeconfigRaw, namespace, releaseName string) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		cmd := exec.CommandContext(ctx, "helm", "test", releaseName,
			"-n", namespace, "--kubeconfig", kp)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm test failed: %s", output)
		}
		return nil
	})
	return output, err
}

// RepoAdd registers a standard HTTP helm repository or an OCI registry.
// For ECR it exchanges AWS credentials for a short-lived Docker token first.
func (s *Service) RepoAdd(ctx context.Context, name, url, providerID string, creds map[string]string) error {
	spec, ok := providers.Find(providerID)
	if !ok {
		return fmt.Errorf("unknown provider %q", providerID)
	}
	if spec.IsOCI {
		return s.ecrLogin(ctx, creds)
	}
	args, err := providers.RepoAddArgs(name, url, providerID, creds)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "helm", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helm repo add: %s", string(out))
	}
	return nil
}

// RepoRemove removes a helm repository by name.
func (s *Service) RepoRemove(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "helm", "repo", "remove", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helm repo remove: %s", string(out))
	}
	return nil
}

// RepoUpdate runs `helm repo update` for a single named repository.
func (s *Service) RepoUpdate(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "helm", "repo", "update", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helm repo update: %s", string(out))
	}
	return nil
}

// RepoTest verifies connectivity by attempting repo add in dry-run fashion.
// For standard repos it runs `helm repo add --no-update`; for ECR it calls GetAuthorizationToken.
func (s *Service) RepoTest(ctx context.Context, name, url, providerID string, creds map[string]string) error {
	spec, ok := providers.Find(providerID)
	if !ok {
		return fmt.Errorf("unknown provider %q", providerID)
	}
	if spec.IsOCI {
		_, err := providers.ECRToken(ctx, creds)
		return err
	}
	args, err := providers.RepoAddArgs(name+"--test", url, providerID, creds)
	if err != nil {
		return err
	}
	args = append(args, "--no-update")
	cmd := exec.CommandContext(ctx, "helm", args...)
	out, err := cmd.CombinedOutput()
	// clean up test repo regardless of result
	_ = exec.CommandContext(ctx, "helm", "repo", "remove", name+"--test").Run()
	if err != nil {
		return fmt.Errorf("helm repo test: %s", string(out))
	}
	return nil
}

func (s *Service) ecrLogin(ctx context.Context, creds map[string]string) error {
	token, err := providers.ECRToken(ctx, creds)
	if err != nil {
		return err
	}
	host := providers.ECRLoginHost(creds)
	cmd := exec.CommandContext(ctx, "helm", "registry", "login", host,
		"--username", "AWS", "--password", token)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("helm registry login: %s", string(out))
	}
	return nil
}

// parseChart splits "nginx-1.2.3" into ("nginx", "1.2.3").
func parseChart(chart string) (name, version string) {
	for i := len(chart) - 1; i >= 0; i-- {
		if chart[i] == '-' {
			return chart[:i], chart[i+1:]
		}
	}
	return chart, ""
}

func parseHelmTime(s string) time.Time {
	for _, f := range []string{
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		time.RFC3339,
	} {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			return t
		}
	}
	return time.Now()
}

// Install deploys a new Helm release. Uses `helm upgrade --install` so it is
// idempotent — safe to call even if the release already exists.
func (s *Service) Install(ctx context.Context, kubeconfigRaw, namespace, releaseName, chart, version string, values map[string]string) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		args := []string{
			"upgrade", releaseName, chart,
			"-n", namespace, "--install", "--create-namespace",
			"--kubeconfig", kp,
			"--timeout", "5m",
		}
		if version != "" {
			args = append(args, "--version", version)
		}
		for k, v := range values {
			args = append(args, "--set", k+"="+v)
		}
		cmd := exec.CommandContext(ctx, "helm", args...)
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm install: %s", output)
		}
		return nil
	})
	return output, err
}

// Uninstall removes a Helm release from the cluster.
func (s *Service) Uninstall(ctx context.Context, kubeconfigRaw, namespace, releaseName string) (string, error) {
	var output string
	err := s.withKubeconfig(kubeconfigRaw, func(kp string) error {
		cmd := exec.CommandContext(ctx, "helm", "uninstall", releaseName,
			"-n", namespace, "--kubeconfig", kp, "--timeout", "5m")
		out, err := cmd.CombinedOutput()
		output = string(out)
		if err != nil {
			return fmt.Errorf("helm uninstall: %s", output)
		}
		return nil
	})
	return output, err
}

// helmIndexFile mirrors the structure of a Helm repository index.yaml.
type helmIndexFile struct {
	Entries map[string][]helmChartEntry `yaml:"entries"`
}

type helmChartEntry struct {
	Name        string   `yaml:"name"`
	Version     string   `yaml:"version"`
	AppVersion  string   `yaml:"appVersion"`
	Description string   `yaml:"description"`
	Keywords    []string `yaml:"keywords"`
	Icon        string   `yaml:"icon"`
}

// indexClient is a dedicated HTTP client for fetching repo index files.
// A short timeout prevents a slow registry from blocking the request handler.
var indexClient = &http.Client{Timeout: 15 * time.Second}

// ListCharts fetches and parses the index.yaml of a Helm HTTP repository,
// returning a flat list of chart versions grouped by chart name.
func (s *Service) ListCharts(ctx context.Context, repoURL, username, password string) ([]types.ChartInfo, error) {
	indexURL := strings.TrimRight(repoURL, "/") + "/index.yaml"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build index request: %w", err)
	}
	if username != "" {
		req.SetBasicAuth(username, password)
	}
	req.Header.Set("Accept", "application/yaml, text/yaml, */*")

	resp, err := indexClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch index.yaml: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, fmt.Errorf("repository returned %d — check credentials", resp.StatusCode)
	case http.StatusNotFound:
		return nil, fmt.Errorf("index.yaml not found at %s — verify the repository URL", indexURL)
	default:
		return nil, fmt.Errorf("index.yaml returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20)) // cap at 32 MB
	if err != nil {
		return nil, fmt.Errorf("read index.yaml: %w", err)
	}

	var idx helmIndexFile
	if err := yaml.Unmarshal(body, &idx); err != nil {
		return nil, fmt.Errorf("parse index.yaml: %w", err)
	}
	if len(idx.Entries) == 0 {
		return nil, fmt.Errorf("index.yaml parsed but contains no chart entries — the repository may be empty or the URL may point to the wrong path")
	}

	var out []types.ChartInfo
	for chartName, versions := range idx.Entries {
		for _, v := range versions {
			out = append(out, types.ChartInfo{
				Name:        chartName,
				Version:     v.Version,
				AppVersion:  v.AppVersion,
				Description: v.Description,
				Keywords:    v.Keywords,
				Icon:        v.Icon,
			})
		}
	}
	return out, nil
}
