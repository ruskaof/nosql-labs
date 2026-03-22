package handler

import (
	"context"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/db/session"
	"time"

	"github.com/redis/go-redis/v9"
)

type SessionHandler struct {
	config *config.ApplicationConfig
	store  session.SessionStore
}

func NewSessionHandler(config *config.ApplicationConfig, store session.SessionStore) *SessionHandler {
	return &SessionHandler{config: config, store: store}
}

const sessionCookieName = "X-Session-Id"

func (h *SessionHandler) SessionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := context.Background()
	ttl := time.Duration(h.config.AppUserSessionTTL) * time.Second

	existing, err := h.getExistingSession(ctx, r, ttl)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if existing != nil {
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

func (h *SessionHandler) getExistingSession(ctx context.Context, r *http.Request, ttl time.Duration) (*string, error) {
	c, err := r.Cookie(sessionCookieName)
	if errors.Is(err, http.ErrNoCookie) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	exists, err := h.store.Exists(ctx, c.Value)
	if err != nil || !exists {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	if err := h.store.Update(ctx, c.Value, ttl); err != nil {
		return nil, err
	}
	return &c.Value, nil
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

func (h *SessionHandler) SetSessionID(w http.ResponseWriter, sessionID string) {
	h.setSessionCookie(w, sessionID)
}

func (h *SessionHandler) RefreshSessionForPost(w http.ResponseWriter, r *http.Request) error {
	ctx := context.Background()
	ttl := time.Duration(h.config.AppUserSessionTTL) * time.Second
	c, err := r.Cookie(sessionCookieName)
	if err != nil || c.Value == "" {
		return err
	}
	exists, err := h.store.Exists(ctx, c.Value)
	if err != nil || !exists {
		return err
	}
	if err := h.store.Update(ctx, c.Value, ttl); err != nil {
		return err
	}
	h.setSessionCookie(w, c.Value)
	return nil
}

func (h *SessionHandler) WriteSessionCookie(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		h.setSessionCookie(w, c.Value)
	}
}
