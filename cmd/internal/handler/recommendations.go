package handler

import (
	"encoding/json"
	"net/http"
)

func (h *HttpHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID, exists, err := h.sessionStore.GetUserID(ctx, c.Value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !exists || userID == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	events, err := h.recommendationService.GetRecommendations(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
}
