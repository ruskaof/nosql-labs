package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/db/user"
	"nosql-labs/cmd/internal/model"
	"time"
)

func (h *HttpHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	ttl := time.Duration(h.cfg.AppUserSessionTTL) * time.Second

	var body model.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}

	if !isStringValid(body.FullName) {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("full_name"))
		return
	}
	if !isStringValid(body.Username) {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("username"))
		return
	}
	if !isStringValid(body.Password) {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("password"))
		return
	}

	c, err := r.Cookie(sessionCookieName)
	if !errors.Is(err, http.ErrNoCookie) && err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c != nil && c.Value != "" {
		exists, err := h.sessionStore.Exists(ctx, c.Value)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	passwordHash, err := HashPassword(*body.Password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	userID, err := h.userStore.Create(ctx, *body.FullName, *body.Username, passwordHash)
	if err != nil {
		if errors.Is(err, user.ErrExists) {
			writeJSONMessage(w, http.StatusConflict, "user already exists")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	sessionID, err := session.GenerateID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if c != nil && c.Value != "" {
		if err := h.sessionStore.Delete(ctx, c.Value); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if err := h.sessionStore.SaveWithUser(ctx, sessionID, userID.Hex(), ttl); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.SetSessionID(w, sessionID)
	w.WriteHeader(http.StatusCreated)
}
