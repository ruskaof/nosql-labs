package handler

import (
	"context"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/session"
	"time"
)

type SessionHandler struct {
	config *config.ApplicationConfig
	store  session.Store
}

func NewSessionHandler(config *config.ApplicationConfig, store session.Store) *SessionHandler {
	return &SessionHandler{config: config, store: store}
}

const sessionCookieName = "x-session-id"

func (h *SessionHandler) SessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	ttl := time.Duration(h.config.AppUserSessionTTL) * time.Second

	if existing := h.getExistingSession(ctx, r, ttl); existing != nil {
		h.setSessionCookie(w, *existing)
		w.WriteHeader(http.StatusOK)
		return
	}

	sessionID, err := session.GenerateID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.store.Save(ctx, sessionID, ttl); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.setSessionCookie(w, sessionID)
	w.WriteHeader(http.StatusCreated)
}

func (h *SessionHandler) getExistingSession(ctx context.Context, r *http.Request, ttl time.Duration) *string {
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return nil
	}
	exists, err := h.store.Exists(ctx, c.Value)
	if err != nil || !exists {
		return nil
	}
	if err := h.store.Update(ctx, c.Value, ttl); err != nil {
		return nil
	}
	return &c.Value
}

func (h *SessionHandler) setSessionCookie(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   h.config.AppUserSessionTTL,
		HttpOnly: true,
	})
}

func (h *SessionHandler) EchoSessionCookie(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		h.setSessionCookie(w, c.Value)
	}
}
