package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func listAuditEventsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := queryInt(r, "page", 1)
		limit := queryInt(r, "limit", 50)
		res, err := svc.ListAuditEvents(r.Context(), middleware.OrgID(r.Context()), page, limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		setPaginationHeaders(w, res.Meta.Page, res.Meta.Limit, res.Meta.Total, res.Meta.Pages)
		writeJSON(w, http.StatusOK, res.Items)
	}
}

func getAuditEventHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		event, err := svc.GetAuditEvent(r.Context(), middleware.OrgID(r.Context()), chi.URLParam(r, "eventId"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Event not found"})
			return
		}
		writeJSON(w, http.StatusOK, event)
	}
}

func auditStatsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		if period == "" {
			period = "24 hours"
		}
		stats, err := svc.GetAuditStats(r.Context(), middleware.OrgID(r.Context()), period)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, stats)
	}
}

func complianceHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.ListCompliance(r.Context(), middleware.OrgID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}
