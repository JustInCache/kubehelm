package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/ankushko/k8s-project-revamp/internal/middleware"
	"github.com/ankushko/k8s-project-revamp/internal/service"
)

func listChannelsHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.ListChannels(r.Context(), middleware.OrgID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func createChannelHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Type == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name and type required"})
			return
		}
		item, err := svc.CreateChannel(r.Context(), middleware.OrgID(r.Context()), body.Name, body.Type)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, item)
	}
}

func updateChannelHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Enabled *bool `json:"enabled"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		_ = json.NewDecoder(r.Body).Decode(&body)
		item, err := svc.UpdateChannel(r.Context(), middleware.OrgID(r.Context()), chi.URLParam(r, "channelId"), body.Enabled)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Channel not found"})
			return
		}
		writeJSON(w, http.StatusOK, item)
	}
}

func deleteChannelHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := svc.DeleteChannel(r.Context(), middleware.OrgID(r.Context()), chi.URLParam(r, "channelId")); err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"error": "Channel not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"message": "Channel deleted"})
	}
}

func testChannelHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"message": "Test notification sent"})
	}
}

func listRulesHandler(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		items, err := svc.ListRules(r.Context(), middleware.OrgID(r.Context()))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, items)
	}
}

func createRuleHandler(svc *service.Service) http.HandlerFunc {
	type req struct {
		Name       string         `json:"name"`
		Events     []string       `json:"events"`
		ChannelIDs []string       `json:"channelIds"`
		Filters    map[string]any `json:"filters"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var body req
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || len(body.Events) == 0 || len(body.ChannelIDs) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "name, events, channelIds required"})
			return
		}
		item, err := svc.CreateRule(r.Context(), middleware.OrgID(r.Context()), body.Name, body.Events, body.ChannelIDs, body.Filters)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, item)
	}
}
