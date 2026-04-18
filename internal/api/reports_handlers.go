package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func listReportsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.ListReports(r.Context(), middleware.OrgID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func createReportHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Name    string         `json:"name"`
		Type    string         `json:"type"`
		Format  string         `json:"format"`
		Filters map[string]any `json:"filters"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Type == "" || body.Format == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name, type, format required"})
			return
		}
		item, err := svc.CreateReport(r.Context(), middleware.OrgID(r.Context()), middleware.UserID(r.Context()), body.Name, body.Type, body.Format, body.Filters)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusAccepted, item)
	}
}

func reportDownloadHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		report, err := svc.GetReport(r.Context(), middleware.OrgID(r.Context()), chi.URLParam(r, "reportId"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Report not found"})
			return
		}
		if report.Status != "completed" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "Report is not ready yet"})
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Disposition", `attachment; filename="report-`+report.ID+`.txt"`)
		_, _ = w.Write([]byte("report download placeholder"))
	}
}
