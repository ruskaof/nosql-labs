package handler

import (
	"net/http"
	"nosql-labs/cmd/internal/config"
	"nosql-labs/cmd/internal/session"
)

type HttpHandler struct {
	sessionHandler *SessionHandler
}

func NewHttpHandler(config *config.ApplicationConfig, store session.Store) *HttpHandler {
	return &HttpHandler{sessionHandler: NewSessionHandler(config, store)}
}

func (h *HttpHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	h.sessionHandler.EchoSessionCookie(w, r)
	w.Write([]byte("{\"status\":\"ok\"}"))
}

func (h *HttpHandler) SessionHandler(w http.ResponseWriter, r *http.Request) {
	h.sessionHandler.SessionHandler(w, r)
}
