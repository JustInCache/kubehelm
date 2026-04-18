package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func getSettingsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		org, users, err := svc.GetSettings(r.Context(), middleware.OrgID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"organization": org, "users": users})
	}
}

func updateOrganizationHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Name     string         `json:"name"`
		Settings map[string]any `json:"settings"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		org, err := svc.UpdateOrganization(r.Context(), middleware.OrgID(r.Context()), body.Name, body.Settings)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, org)
	}
}

func inviteUserHandler() http.HandlerFunc {
	type req struct {
		Email string `json:"email"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		writeJSON(w, http.StatusOK, map[string]any{"message": "Invitation sent to " + body.Email})
	}
}

func updateUserRoleHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Role string `json:"role"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		user, err := svc.UpdateUserRole(r.Context(), middleware.OrgID(r.Context()), chi.URLParam(r, "userId"), body.Role)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "User not found"})
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}
