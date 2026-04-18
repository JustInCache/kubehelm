package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

// releaseParam reads and URL-decodes the releaseId path parameter.
// Release IDs for live clusters are "clusterID/namespace/name" which must
// be percent-encoded as a single path segment by the client.
func releaseParam(r *http.Request) string {
	raw := chi.URLParam(r, "releaseId")
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return raw
	}
	return decoded
}

func listReleasesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 25)
		namespace := r.URL.Query().Get("namespace")
		search := r.URL.Query().Get("search")
		sortBy := r.URL.Query().Get("sortBy")
		sortOrder := r.URL.Query().Get("sortOrder")
		res, err := svc.ListReleases(r.Context(), middleware.OrgID(r.Context()), namespace, page, limit, search, sortBy, sortOrder)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func listReleaseHistoryHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 20)
		res, err := svc.ListReleaseHistory(r.Context(), middleware.OrgID(r.Context()), releaseID, page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func releaseManifestHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		rev := queryInt(r, "revision", 0)
		if rev == 0 {
			parsed, err := strconv.Atoi(chi.URLParam(r, "revision"))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid revision"})
				return
			}
			rev = parsed
		}
		manifest, err := svc.GetManifest(r.Context(), middleware.OrgID(r.Context()), releaseID, rev)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "revision not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"manifest": manifest})
	}
}

func releaseDiffHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		revA := queryInt(r, "revA", 0)
		revB := queryInt(r, "revB", 0)
		if revA == 0 || revB == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "revA and revB query params required"})
			return
		}
		diff, err := svc.Diff(r.Context(), middleware.OrgID(r.Context()), releaseID, revA, revB)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"manifestDiff": diff,
			"valuesDiff":   diff,
			"revA":         revA,
			"revB":         revB,
		})
	}
}

func listDriftHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 50)
		res, err := svc.ListDrift(r.Context(), middleware.OrgID(r.Context()), page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func listApprovalsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 100)
		status := r.URL.Query().Get("status")
		res, err := svc.ListApprovals(r.Context(), middleware.OrgID(r.Context()), status, page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func dryRunHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Chart   string `json:"chart"`
		Version string `json:"version"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		manifest, err := svc.DryRun(r.Context(), middleware.OrgID(r.Context()), releaseID, body.Chart, body.Version)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"manifest": manifest,
			"diff": map[string]any{
				"hunks": []any{},
				"stats": map[string]int{"added": 0, "removed": 0, "unchanged": 0},
			},
		})
	}
}

func upgradeHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Chart   string            `json:"chart"`
		Version string            `json:"version"`
		Values  map[string]string `json:"values"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		orgID := middleware.OrgID(r.Context())
		username := middleware.UserEmail(r.Context())
		output, err := svc.UpgradeRelease(r.Context(), orgID, releaseID, body.Chart, body.Version, username, body.Values)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "output": output})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"output":  output,
			"message": "Upgrade completed successfully",
		})
	}
}

func rollbackHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Revision int `json:"revision"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		orgID := middleware.OrgID(r.Context())
		username := middleware.UserEmail(r.Context())
		output, err := svc.RollbackRelease(r.Context(), orgID, releaseID, body.Revision, username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error(), "output": output})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"output":  output,
			"message": "Rollback completed successfully",
		})
	}
}

func releaseTestHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		output, err := svc.TestRelease(r.Context(), middleware.OrgID(r.Context()), releaseID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success": true,
			"output":  output,
		})
	}
}

func releaseValuesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		values, err := svc.GetReleaseValues(r.Context(), middleware.OrgID(r.Context()), releaseID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"values": values})
	}
}

func installReleaseHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		ClusterID   string            `json:"clusterId"`
		Namespace   string            `json:"namespace"`
		ReleaseName string            `json:"releaseName"`
		Chart       string            `json:"chart"`
		Version     string            `json:"version"`
		Values      map[string]string `json:"values"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request"})
			return
		}
		if body.ClusterID == "" || body.Namespace == "" || body.ReleaseName == "" || body.Chart == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "clusterId, namespace, releaseName, chart are required"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		username := middleware.UserEmail(r.Context())
		output, err := svc.InstallRelease(r.Context(), orgID, body.ClusterID, body.Namespace, body.ReleaseName, body.Chart, body.Version, username, body.Values)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error(), "output": output})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"success": true, "output": output})
	}
}

func uninstallReleaseHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		releaseID := releaseParam(r)
		orgID := middleware.OrgID(r.Context())
		username := middleware.UserEmail(r.Context())
		output, err := svc.UninstallRelease(r.Context(), orgID, releaseID, username)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": true, "output": output})
	}
}

func listChartsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		repoID := r.URL.Query().Get("repoId")
		if repoID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "repoId query param required"})
			return
		}
		orgID := middleware.OrgID(r.Context())
		charts, err := svc.ListCharts(r.Context(), orgID, repoID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, charts)
	}
}

func approveHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		approvalID := chi.URLParam(r, "approvalId")
		approval, err := svc.Approve(r.Context(), approvalID, middleware.UserID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Approval not found"})
			return
		}
		writeJSON(w, http.StatusOK, approval)
	}
}

func rejectHandler(svc *service.Service) http.HandlerFunc {
	type rejectReq struct {
		Reason string `json:"reason"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		approvalID := chi.URLParam(r, "approvalId")
		var body rejectReq
		_ = json.NewDecoder(r.Body).Decode(&body)
		approval, err := svc.Reject(r.Context(), approvalID, middleware.UserID(r.Context()), body.Reason)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Approval not found"})
			return
		}
		writeJSON(w, http.StatusOK, approval)
	}
}
