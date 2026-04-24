package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/model"
	"time"
)

func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	ttl := time.Duration(h.cfg.AppUserSessionTTL) * time.Second

	var body model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}
	if !validateRequiredFields(w,
		requiredField{name: "username", value: body.Username},
		requiredField{name: "password", value: body.Password},
	) {
		return
	}

	rec, err := h.userStore.FindByUsername(ctx, *body.Username)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rec == nil || !CheckPassword(rec.PasswordHash, *body.Password) {
		writeJSONMessage(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	userHex := rec.ID.Hex()
	c, err := r.Cookie(sessionCookieName)
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c != nil && c.Value != "" {
		exists, err := h.sessionStore.Exists(ctx, c.Value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists {
			if err := h.sessionStore.SetUserID(ctx, c.Value, userHex, ttl); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			h.sessionHandler.SetSessionID(w, c.Value)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	sid, err := session.GenerateID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.sessionStore.SaveWithUser(ctx, sid, userHex, ttl); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.SetSessionID(w, sid)
	w.WriteHeader(http.StatusNoContent)
}

func (h *HttpHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = h.sessionStore.Delete(ctx, c.Value)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	expireSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func expireSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
	})
}
