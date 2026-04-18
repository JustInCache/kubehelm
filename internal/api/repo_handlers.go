package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/helm/providers"
	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

// GET /api/helm/providers — return all built-in provider specs.
func listProvidersHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, providers.All)
	}
}

// GET /api/helm/providers/enabled — return IDs of org's installed providers.
func listEnabledProvidersHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgID(r.Context())
		ids, err := svc.GetEnabledProviders(r.Context(), orgID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, ids)
	}
}

// POST /api/helm/providers/install — { "providerId": "harbor" }
func installProviderHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProviderID string `json:"providerId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ProviderID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "providerId required"})
			return
		}
		if _, ok := providers.Find(body.ProviderID); !ok {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown provider"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		enabled, _ := svc.GetEnabledProviders(r.Context(), orgID)
		for _, id := range enabled {
			if id == body.ProviderID {
				writeJSON(w, http.StatusOK, map[string]any{"message": "already installed"})
				return
			}
		}
		enabled = append(enabled, body.ProviderID)
		if err := svc.SetEnabledProviders(r.Context(), orgID, enabled); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "installed", "enabledProviders": enabled})
	}
}

// POST /api/helm/providers/uninstall — { "providerId": "harbor" }
func uninstallProviderHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ProviderID string `json:"providerId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ProviderID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "providerId required"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		enabled, _ := svc.GetEnabledProviders(r.Context(), orgID)
		filtered := enabled[:0]
		for _, id := range enabled {
			if id != body.ProviderID {
				filtered = append(filtered, id)
			}
		}
		if err := svc.SetEnabledProviders(r.Context(), orgID, filtered); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "uninstalled", "enabledProviders": filtered})
	}
}

// GET /api/helm/repositories
func listHelmRepositoriesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgID(r.Context())
		repos, err := svc.ListHelmRepositories(r.Context(), orgID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, repos)
	}
}

// POST /api/helm/repositories
func addHelmRepositoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name        string            `json:"name"`
			URL         string            `json:"url"`
			ProviderID  string            `json:"providerId"`
			Credentials map[string]string `json:"credentials"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request"})
			return
		}
		if body.Name == "" || body.ProviderID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name and providerId required"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		repo, err := svc.AddHelmRepository(r.Context(), orgID, body.Name, body.URL, body.ProviderID, body.Credentials)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, repo)
	}
}

// PUT /api/helm/repositories/{repoId}
func updateHelmRepositoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			URL         string            `json:"url"`
			Credentials map[string]string `json:"credentials"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		repoID := chi.URLParam(r, "repoId")
		repo, err := svc.UpdateHelmRepository(r.Context(), orgID, repoID, body.URL, body.Credentials)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, repo)
	}
}

// DELETE /api/helm/repositories/{repoId}
func deleteHelmRepositoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgID(r.Context())
		repoID := chi.URLParam(r, "repoId")
		if err := svc.RemoveHelmRepository(r.Context(), orgID, repoID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// POST /api/helm/repositories/{repoId}/refresh
func refreshHelmRepositoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgID(r.Context())
		repoID := chi.URLParam(r, "repoId")
		if err := svc.RefreshHelmRepository(r.Context(), orgID, repoID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "refreshed"})
	}
}

// POST /api/helm/repositories/{repoId}/test
func testHelmRepositoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		orgID := middleware.OrgID(r.Context())
		repoID := chi.URLParam(r, "repoId")
		if err := svc.TestHelmRepository(r.Context(), orgID, repoID); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}
