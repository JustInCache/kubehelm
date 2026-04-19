package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
	"github.com/ankushko/k8s-project-revamp/internal/version"
)

func NewRouter(svc *service.Service, logger *slog.Logger, cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(corsMiddleware)
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(chimw.Timeout(time.Duration(cfg.ReadTimeoutMS) * time.Millisecond))
	r.Use(middleware.RequestLogger(logger))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "ts": time.Now().UTC().Format(time.RFC3339)})
	})
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, version.Get())
	})
	r.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if !svc.Ready(r.Context()) {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"status": "ready"})
	})
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/api", func(api chi.Router) {
		api.Route("/auth", func(auth chi.Router) {
			auth.Post("/login", loginHandler(svc))
			auth.Post("/refresh", refreshHandler())
			auth.Post("/register", registerHandler())
			auth.With(middleware.Auth(cfg.JWTSecret)).Get("/me", meHandler(svc))
		})

		api.Group(func(protected chi.Router) {
			protected.Use(middleware.Auth(cfg.JWTSecret))
			protected.Get("/stream", sseHandler(svc))

			protected.Get("/clusters", listClustersHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Post("/clusters", createClusterHandler(svc))
			protected.Get("/clusters/{clusterId}", clusterDetailHandler(svc))
			protected.With(middleware.RequireRole("admin")).Delete("/clusters/{clusterId}", deleteClusterHandler(svc))
			protected.Get("/clusters/{clusterId}/health", clusterHealthHandler(svc))
			protected.Get("/clusters/{clusterId}/nodes", clusterNodesHandler(svc))
			protected.Get("/clusters/{clusterId}/namespaces", clusterNamespacesHandler(svc))
			protected.Post("/clusters/test-connection", clusterTestConnectionHandler(svc))
			protected.Post("/clusters/{clusterId}/test-connection", clusterTestConnectionHandler(svc))

			protected.Get("/helm/releases", listReleasesHandler(svc))
			protected.Get("/helm/charts", listChartsHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Post("/helm/releases", installReleaseHandler(svc))
			protected.Get("/helm/releases/{releaseId}/history", listReleaseHistoryHandler(svc))
			protected.Get("/helm/releases/{releaseId}/history/{revision}/manifest", releaseManifestHandler(svc))
			protected.Get("/helm/releases/{releaseId}/diff", releaseDiffHandler(svc))
			protected.Get("/helm/releases/{releaseId}/values", releaseValuesHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Delete("/helm/releases/{releaseId}", uninstallReleaseHandler(svc))
			protected.Get("/helm/drift", listDriftHandler(svc))
			protected.Get("/helm/approvals", listApprovalsHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/helm/approvals/{approvalId}/approve", approveHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/helm/approvals/{approvalId}/reject", rejectHandler(svc))
			protected.With(middleware.NewExpensiveRateLimiter(cfg.ExpensiveRateRPS, cfg.ExpensiveRateBurst)).Post("/helm/releases/{releaseId}/dry-run", dryRunHandler(svc))
			protected.With(middleware.NewExpensiveRateLimiter(cfg.ExpensiveRateRPS, cfg.ExpensiveRateBurst)).Post("/helm/releases/{releaseId}/upgrade", upgradeHandler(svc))
			protected.With(middleware.NewExpensiveRateLimiter(cfg.ExpensiveRateRPS, cfg.ExpensiveRateBurst)).Post("/helm/releases/{releaseId}/rollback", rollbackHandler(svc))
			protected.With(middleware.NewExpensiveRateLimiter(cfg.ExpensiveRateRPS, cfg.ExpensiveRateBurst)).Post("/helm/releases/{releaseId}/test", releaseTestHandler(svc))

			// Helm provider plugins
			protected.Get("/helm/providers", listProvidersHandler())
			protected.Get("/helm/providers/enabled", listEnabledProvidersHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/helm/providers/install", installProviderHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/helm/providers/uninstall", uninstallProviderHandler(svc))

			// Helm repositories
			protected.Get("/helm/repositories", listHelmRepositoriesHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Post("/helm/repositories", addHelmRepositoryHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Put("/helm/repositories/{repoId}", updateHelmRepositoryHandler(svc))
			protected.With(middleware.RequireRole("admin")).Delete("/helm/repositories/{repoId}", deleteHelmRepositoryHandler(svc))
			protected.Post("/helm/repositories/{repoId}/refresh", refreshHelmRepositoryHandler(svc))
			protected.Post("/helm/repositories/{repoId}/test", testHelmRepositoryHandler(svc))

			protected.Get("/audit/events", listAuditEventsHandler(svc))
			protected.Get("/audit/events/{eventId}", getAuditEventHandler(svc))
			protected.Get("/audit/stats", auditStatsHandler(svc))
			protected.Get("/audit/compliance", complianceHandler(svc))

			protected.Get("/notifications/channels", listChannelsHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/notifications/channels", createChannelHandler(svc))
			protected.With(middleware.RequireRole("admin")).Put("/notifications/channels/{channelId}", updateChannelHandler(svc))
			protected.With(middleware.RequireRole("admin")).Delete("/notifications/channels/{channelId}", deleteChannelHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/notifications/channels/{channelId}/test", testChannelHandler())
			protected.Get("/notifications/rules", listRulesHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/notifications/rules", createRuleHandler(svc))

			protected.Get("/reports", listReportsHandler(svc))
			protected.With(middleware.RequireRole("admin", "sre")).Post("/reports", createReportHandler(svc))
			protected.Get("/reports/{reportId}/download", reportDownloadHandler(svc))

			protected.Get("/settings", getSettingsHandler(svc))
			protected.With(middleware.RequireRole("admin")).Put("/settings/organization", updateOrganizationHandler(svc))
			protected.With(middleware.RequireRole("admin")).Post("/settings/users/invite", inviteUserHandler())
			protected.With(middleware.RequireRole("admin")).Put("/settings/users/{userId}/role", updateUserRoleHandler(svc))
		})
	})

	return r
}

func queryInt(r *http.Request, key string, fallback int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func setPaginationHeaders(w http.ResponseWriter, page, limit, total, pages int) {
	w.Header().Set("X-Page", strconv.Itoa(page))
	w.Header().Set("X-Limit", strconv.Itoa(limit))
	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	w.Header().Set("X-Total-Pages", strconv.Itoa(pages))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
