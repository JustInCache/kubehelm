package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ankushko/k8s-project-revamp/internal/cache"
	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/helm"
	k8sclient "github.com/ankushko/k8s-project-revamp/internal/k8s"
	"github.com/ankushko/k8s-project-revamp/internal/service"
	"github.com/ankushko/k8s-project-revamp/internal/store"
)

func TestLoginAndHealthz(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret"
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	svc := service.NewService(
		store.NewMemoryRepository(),
		cache.NewRedis("localhost:6379", "", 0),
		cache.NewInMemoryCache(1*time.Minute),
		cache.NewCoalescer(),
		logger,
		cfg,
		helm.NewService(),
		k8sclient.NewManager(),
	)
	router := NewRouter(svc, logger, cfg)

	healthReq := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	healthRec := httptest.NewRecorder()
	router.ServeHTTP(healthRec, healthReq)
	require.Equal(t, http.StatusOK, healthRec.Code)

	loginBody := map[string]string{"email": "admin@kubeaudit.io", "password": "Admin@123"}
	b, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(b))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)
	require.Contains(t, loginRec.Body.String(), "token")
}

func TestProtectedClustersEndpoint(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret"
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	svc := service.NewService(
		store.NewMemoryRepository(),
		cache.NewRedis("localhost:6379", "", 0),
		cache.NewInMemoryCache(1*time.Minute),
		cache.NewCoalescer(),
		logger,
		cfg,
	)
	router := NewRouter(svc, logger, cfg)

	loginBody := map[string]string{"email": "admin@kubeaudit.io", "password": "Admin@123"}
	b, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(b))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var loginRes map[string]any
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginRes))
	token, _ := loginRes["token"].(string)
	require.NotEmpty(t, token)

	req := httptest.NewRequest(http.MethodGet, "/api/clusters", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestHelmCompatibilityEndpoints(t *testing.T) {
	cfg := config.Load()
	cfg.JWTSecret = "test-secret"
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))
	svc := service.NewService(
		store.NewMemoryRepository(),
		cache.NewRedis("localhost:6379", "", 0),
		cache.NewInMemoryCache(1*time.Minute),
		cache.NewCoalescer(),
		logger,
		cfg,
	)
	router := NewRouter(svc, logger, cfg)

	loginBody := map[string]string{"email": "admin@kubeaudit.io", "password": "Admin@123"}
	b, _ := json.Marshal(loginBody)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(b))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	var loginRes map[string]any
	require.NoError(t, json.Unmarshal(loginRec.Body.Bytes(), &loginRes))
	token, _ := loginRes["token"].(string)
	require.NotEmpty(t, token)

	historyReq := httptest.NewRequest(http.MethodGet, "/api/helm/releases/22222222-2222-2222-2222-222222222222/history", nil)
	historyReq.Header.Set("Authorization", "Bearer "+token)
	historyRec := httptest.NewRecorder()
	router.ServeHTTP(historyRec, historyReq)
	require.Equal(t, http.StatusOK, historyRec.Code)

	diffReq := httptest.NewRequest(http.MethodGet, "/api/helm/releases/22222222-2222-2222-2222-222222222222/diff?revA=13&revB=14", nil)
	diffReq.Header.Set("Authorization", "Bearer "+token)
	diffRec := httptest.NewRecorder()
	router.ServeHTTP(diffRec, diffReq)
	require.Equal(t, http.StatusOK, diffRec.Code)

	approvalsReq := httptest.NewRequest(http.MethodGet, "/api/helm/approvals", nil)
	approvalsReq.Header.Set("Authorization", "Bearer "+token)
	approvalsRec := httptest.NewRecorder()
	router.ServeHTTP(approvalsRec, approvalsReq)
	require.Equal(t, http.StatusOK, approvalsRec.Code)
}
