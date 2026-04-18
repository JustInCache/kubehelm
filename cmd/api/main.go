package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ankushko/k8s-project-revamp/internal/api"
	"github.com/ankushko/k8s-project-revamp/internal/cache"
	"github.com/ankushko/k8s-project-revamp/internal/config"
	"github.com/ankushko/k8s-project-revamp/internal/helm"
	k8sclient "github.com/ankushko/k8s-project-revamp/internal/k8s"
	"github.com/ankushko/k8s-project-revamp/internal/service"
	"github.com/ankushko/k8s-project-revamp/internal/store"
)

// ensureNoProxy guarantees that private/loopback network ranges are never
// routed through a corporate or system HTTP proxy. Without this, Go's default
// http.ProxyFromEnvironment sends Kubernetes API calls (e.g. to minikube at
// 192.168.x.x) through whatever HTTPS_PROXY is set, causing connection failures.
func ensureNoProxy() {
	const privateRanges = "localhost,127.0.0.1,::1,10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,.svc,.cluster.local,.local"
	for _, key := range []string{"NO_PROXY", "no_proxy"} {
		existing := os.Getenv(key)
		if existing == "" {
			os.Setenv(key, privateRanges)
			continue
		}
		// Append private ranges that aren't already present
		extra := []string{}
		for _, r := range strings.Split(privateRanges, ",") {
			if !strings.Contains(existing, r) {
				extra = append(extra, r)
			}
		}
		if len(extra) > 0 {
			os.Setenv(key, existing+","+strings.Join(extra, ","))
		}
	}
}

func main() {
	ensureNoProxy()
	cfg := config.Load()
	logger := config.NewLogger(cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	redisCache := cache.NewRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)
	defer redisCache.Close()

	memCache := cache.NewInMemoryCache(30 * time.Second)
	defer memCache.Close()

	coalescer := cache.NewCoalescer()

	var repo store.Repository = store.NewMemoryRepository()
	if !cfg.UseMemoryStore {
		pgRepo, err := store.NewPostgresRepository(ctx, cfg, logger)
		if err != nil {
			logger.Warn("postgres unavailable, falling back to memory store", slog.String("error", err.Error()))
		} else {
			repo = pgRepo
			defer pgRepo.Close()
			logger.Info("postgres store enabled")
		}
	}

	helmSvc := helm.NewService()
	k8sMgr := k8sclient.NewManager()

	if helmSvc.IsAvailable() {
		logger.Info("helm CLI detected — live release data enabled")
	} else {
		logger.Warn("helm CLI not found — falling back to stored release data")
	}

	svc := service.NewService(repo, redisCache, memCache, coalescer, logger, cfg, helmSvc, k8sMgr)

	handler := api.NewRouter(svc, logger, cfg)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("api starting", slog.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("shutdown complete")
}
