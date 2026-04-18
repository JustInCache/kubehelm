package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ankushko/k8s-project-revamp/internal/cache"
	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/helm"
	k8sclient "github.com/ankushko/k8s-project-revamp/internal/k8s"
	"github.com/ankushko/k8s-project-revamp/internal/store"
)

func TestLoginAndClusters(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret"
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	svc := NewService(
		store.NewMemoryRepository(),
		cache.NewRedis("localhost:6379", "", 0),
		cache.NewInMemoryCache(1*time.Minute),
		cache.NewCoalescer(),
		logger,
		cfg,
		helm.NewService(),
		k8sclient.NewManager(),
	)

	_, err := svc.Login(context.Background(), "admin@kubeaudit.io", "Admin@123")
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), "admin@kubeaudit.io", "wrong")
	require.Error(t, err)

	clusters, err := svc.ListClusters(context.Background(), "00000000-0000-0000-0000-000000000001", 1, 20)
	require.NoError(t, err)
	require.NotEmpty(t, clusters.Items)
}
