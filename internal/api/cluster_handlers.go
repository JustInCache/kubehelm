package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func listClustersHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 20)
		res, err := svc.ListClusters(r.Context(), middleware.OrgID(r.Context()), page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func createClusterHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Name        string `json:"name"`
		Provider    string `json:"provider"`
		Environment string `json:"environment"`
		Kubeconfig  string `json:"kubeconfig"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		if body.Name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name is required"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		cluster, err := svc.RegisterCluster(r.Context(), orgID, body.Name, body.Provider, body.Environment, body.Kubeconfig)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, cluster)
	}
}

func deleteClusterHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusterID := chi.URLParam(r, "clusterId")
		orgID := middleware.OrgID(r.Context())
		username := middleware.UserEmail(r.Context())
		if err := svc.DeleteCluster(r.Context(), orgID, clusterID, username); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
	}
}

func clusterDetailHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusterID := chi.URLParam(r, "clusterId")
		orgID := middleware.OrgID(r.Context())
		c, err := svc.GetCluster(r.Context(), orgID, clusterID)
		if err != nil || c == nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "cluster not found"})
			return
		}
		health, _ := svc.GetClusterHealth(r.Context(), orgID, clusterID)
		writeJSON(w, http.StatusOK, map[string]any{
			"id":            c.ID,
			"orgId":         c.OrgID,
			"name":          c.Name,
			"provider":      c.Provider,
			"environment":   c.Environment,
			"authType":      c.AuthType,
			"status":        c.Status,
			"serverVersion": c.ServerVersion,
			"lastError":     c.LastError,
			"releaseCount":  c.ReleaseCount,
			"nodeCount":     c.NodeCount,
			"createdAt":     c.CreatedAt,
			"health":        health,
		})
	}
}

func clusterHealthHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusterID := chi.URLParam(r, "clusterId")
		res, err := svc.GetClusterHealth(r.Context(), middleware.OrgID(r.Context()), clusterID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func clusterNodesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusterID := chi.URLParam(r, "clusterId")
		orgID := middleware.OrgID(r.Context())
		nodes, err := svc.GetClusterNodes(r.Context(), orgID, clusterID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, nodes)
	}
}

func clusterNamespacesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clusterID := chi.URLParam(r, "clusterId")
		orgID := middleware.OrgID(r.Context())
		namespaces, err := svc.GetClusterNamespaces(r.Context(), orgID, clusterID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, namespaces)
	}
}

func clusterTestConnectionHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Kubeconfig string `json:"kubeconfig"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Kubeconfig == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "kubeconfig is required"})
			return
		}
		serverVersion, err := svc.TestClusterConnection(r.Context(), body.Kubeconfig)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]any{
				"connected": false,
				"error":     err.Error(),
				"checkedAt": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"connected":     true,
			"serverVersion": serverVersion,
			"checkedAt":     time.Now().UTC().Format(time.RFC3339),
		})
	}
}
