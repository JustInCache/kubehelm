package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func loginHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid request body"})
			return
		}
		res, err := svc.Login(r.Context(), body.Email, body.Password)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "Invalid credentials"})
			return
		}
		writeJSON(w, http.StatusOK, res)
	}
}

func meHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := svc.Me(r.Context(), middleware.UserID(r.Context()))
		if err != nil || user == nil {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": "User not found"})
			return
		}
		writeJSON(w, http.StatusOK, user)
	}
}

func refreshHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"token": uuid.NewString(),
			"exp":   time.Now().Add(24 * time.Hour).Unix(),
		})
	}
}

func registerHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "register not enabled in MVP yet"})
	}
}
