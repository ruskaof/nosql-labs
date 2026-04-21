package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/db/session"
	"nosql-labs/cmd/internal/db/user"
	"nosql-labs/cmd/internal/model"
	"strconv"
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

	if !validateRequiredFields(w,
		requiredField{name: "full_name", value: body.FullName},
		requiredField{name: "username", value: body.Username},
		requiredField{name: "password", value: body.Password},
	) {
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

func (h *HttpHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()

	var limit int64
	var hasLimit bool
	if q.Has("limit") {
		lu, err := strconv.ParseUint(q.Get("limit"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("limit"))
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
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("offset"))
			return
		}
		offset = int64(ou)
	}
	if hasLimit && limit == 0 {
		h.sessionHandler.WriteSessionCookie(w, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"users": []user.PublicUser{},
			"count": 0,
		})
		return
	}

	users, err := h.userStore.List(r.Context(), user.ListFilter{
		ID:     q.Get("id"),
		Name:   q.Get("name"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"users": users,
		"count": len(users),
	})
}

func (h *HttpHandler) GetUserByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	u, err := h.userStore.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	if u == nil {
		writeJSONMessage(w, http.StatusNotFound, "Not found")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(u)
}

func (h *HttpHandler) ListEventsByUserID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	u, err := h.userStore.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	if u == nil {
		writeJSONMessage(w, http.StatusNotFound, "User not found")
		return
	}

	q := r.URL.Query()
	title := q.Get("title")
	eventID := q.Get("id")
	category := q.Get("category")
	city := q.Get("city")

	var limit int64
	var hasLimit bool
	if q.Has("limit") {
		lu, err := strconv.ParseUint(q.Get("limit"), 10, 64)
		if err != nil {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("limit"))
			return
		}
		limit = int64(lu)
		hasLimit = true
	}
	var offset int64
	if q.Has("offset") {
		ou, err := strconv.ParseUint(q.Get("offset"), 10, 64)
		if err != nil {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("offset"))
			return
		}
		offset = int64(ou)
	}

	if category != "" {
		if _, ok := validEventCategories[category]; !ok {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("category"))
			return
		}
	}

	var priceFrom *uint64
	if q.Has("price_from") {
		v, err := strconv.ParseUint(q.Get("price_from"), 10, 64)
		if err != nil {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price_from"))
			return
		}
		priceFrom = &v
	}
	var priceTo *uint64
	if q.Has("price_to") {
		v, err := strconv.ParseUint(q.Get("price_to"), 10, 64)
		if err != nil {
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price_to"))
			return
		}
		priceTo = &v
	}
	if priceFrom != nil && priceTo != nil && *priceFrom > *priceTo {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price_from"))
		return
	}

	dateFromRaw := q.Get("date_from")
	if dateFromRaw == "" {
		dateFromRaw = q.Get("started_date_from")
	}
	dateToRaw := q.Get("date_to")
	if dateToRaw == "" {
		dateToRaw = q.Get("started_date_to")
	}
	dateFrom, dateTo, dateErrField := normalizeDateRange(dateFromRaw, dateToRaw)
	if dateErrField != "" {
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage(dateErrField))
		return
	}

	if hasLimit && limit == 0 {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"events": []event.ListItem{},
			"count":  0,
		})
		return
	}

	events, err := h.eventStore.List(r.Context(), event.ListFilter{
		ID:        eventID,
		Title:     title,
		Category:  category,
		City:      city,
		UserID:    id,
		PriceFrom: priceFrom,
		PriceTo:   priceTo,
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}
