package handler

import (
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/db/user"
)

type HttpHandler struct {
	cfg            *config.ApplicationConfig
	sessionHandler *SessionHandler
	sessionStore   session.SessionStore
	userStore      *user.UserStore
	eventStore     *event.EventStore
}

func NewHttpHandler(cfg *config.ApplicationConfig, sessionStore session.SessionStore, userStore *user.UserStore, eventStore *event.EventStore) *HttpHandler {
	return &HttpHandler{
		cfg:            cfg,
		sessionHandler: NewSessionHandler(cfg, sessionStore),
		sessionStore:   sessionStore,
		userStore:      userStore,
		eventStore:     eventStore,
	}
}

func (h *HttpHandler) WithPostSessionRefresh(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if err := h.sessionHandler.RefreshSessionForPost(w, r); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		next(w, r)
	}
}

func (h *HttpHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.sessionHandler.WriteSessionCookie(w, r)
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func (h *HttpHandler) SessionHandler(w http.ResponseWriter, r *http.Request) {
	h.sessionHandler.SessionHandler(w, r)
}
