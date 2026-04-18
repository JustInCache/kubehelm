package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/ankushko/k8s-project-revamp/internal/types"
)

// Manager provides real Kubernetes API access from raw kubeconfig YAML.
type Manager struct{}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) clientFromKubeconfig(raw string) (kubernetes.Interface, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}
	cfg.Timeout = 8 * time.Second
	return kubernetes.NewForConfig(cfg)
}

// TestConnection verifies connectivity and returns the server version string.
func (m *Manager) TestConnection(ctx context.Context, raw string) (string, error) {
	cs, err := m.clientFromKubeconfig(raw)
	if err != nil {
		return "", err
	}
	ver, err := cs.Discovery().ServerVersion()
	if err != nil {
		return "", fmt.Errorf("connect to cluster: %w", err)
	}
	return ver.GitVersion, nil
}

// GetClusterHealth returns node and pod counts from the cluster.
func (m *Manager) GetClusterHealth(ctx context.Context, raw string) (types.ClusterHealth, error) {
	cs, err := m.clientFromKubeconfig(raw)
	if err != nil {
		return types.ClusterHealth{Error: err.Error()}, nil
	}
	var health types.ClusterHealth
	health.Timestamp = time.Now().UTC().Format(time.RFC3339)

	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err == nil {
		health.Nodes.Total = len(nodes.Items)
		for _, n := range nodes.Items {
			for _, c := range n.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
					health.Nodes.Ready++
				}
			}
		}
		health.Nodes.NotReady = health.Nodes.Total - health.Nodes.Ready
	}

	pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err == nil {
		health.Pods.Total = len(pods.Items)
		for _, p := range pods.Items {
			switch p.Status.Phase {
			case corev1.PodRunning:
				health.Pods.Running++
			case corev1.PodPending:
				health.Pods.Pending++
			case corev1.PodFailed:
				health.Pods.Failed++
			case corev1.PodSucceeded:
				health.Pods.Succeeded++
			}
		}
	}
	return health, nil
}

// GetNodes returns node details including roles, capacity, and status.
func (m *Manager) GetNodes(ctx context.Context, raw string) ([]map[string]any, error) {
	cs, err := m.clientFromKubeconfig(raw)
	if err != nil {
		return nil, err
	}
	nodes, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0, len(nodes.Items))
	for _, n := range nodes.Items {
		status := "NotReady"
		for _, c := range n.Status.Conditions {
			if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue {
				status = "Ready"
			}
		}
		roles := []string{}
		for k := range n.Labels {
			if strings.HasPrefix(k, "node-role.kubernetes.io/") {
				roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/"))
			}
		}
		if len(roles) == 0 {
			roles = []string{"worker"}
		}
		result = append(result, map[string]any{
			"name":    n.Name,
			"status":  status,
			"roles":   roles,
			"cpu":     n.Status.Capacity.Cpu().String(),
			"memory":  n.Status.Capacity.Memory().String(),
			"version": n.Status.NodeInfo.KubeletVersion,
		})
	}
	return result, nil
}

// GetNamespaces lists all namespace names in the cluster.
func (m *Manager) GetNamespaces(ctx context.Context, raw string) ([]string, error) {
	cs, err := m.clientFromKubeconfig(raw)
	if err != nil {
		return nil, err
	}
	nsList, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	names := make([]string, len(nsList.Items))
	for i, ns := range nsList.Items {
		names[i] = ns.Name
	}
	return names, nil
}

func DefaultK8sTimeout() time.Duration { return 8 * time.Second }
