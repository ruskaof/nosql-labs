package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/model"
	"strconv"
)

func (h *HttpHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	var body model.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}

	if !validateRequiredFields(w,
		requiredField{name: "title", value: body.Title},
		requiredField{name: "address", value: body.Address},
		requiredField{name: "started_at", value: body.StartedAt},
		requiredField{name: "finished_at", value: body.FinishedAt},
	) {
		return
	}

	if !validateRFC3339Field(w, "started_at", body.StartedAt) {
		return
	}
	if !validateRFC3339Field(w, "finished_at", body.FinishedAt) {
		return
	}

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

	desc := ""
	if body.Description != nil {
		desc = *body.Description
	}

	id, err := h.eventStore.Create(ctx,
		*body.Title,
		desc,
		*body.Address,
		userID,
		*body.StartedAt,
		*body.FinishedAt,
	)
	if err != nil {
		if errors.Is(err, event.ErrExists) {
			writeJSONMessage(w, http.StatusConflict, "event already exists")
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{"id": id.Hex()}); err != nil {
		return
	}
}

func (h *HttpHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	q := r.URL.Query()

	title := q.Get("title")

	var limit int64
	var hasLimit bool
	if q.Has("limit") {
		lu, err := strconv.ParseUint(q.Get("limit"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidParamMessage("limit"))
			return
		}
		limit = int64(lu)
		hasLimit = true
	}

	var offset int64
	if q.Has("offset") {
		ou, err := strconv.ParseUint(q.Get("offset"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidParamMessage("offset"))
			return
		}
		offset = int64(ou)
	}

	if hasLimit && limit == 0 {
		h.sessionHandler.WriteSessionCookie(w, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []event.ListItem{},
			"count":  0,
		})
		return
	}

	events, err := h.eventStore.List(ctx, event.ListFilter{Title: title, Limit: limit, Offset: offset})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
