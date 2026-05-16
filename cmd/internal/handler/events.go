package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/model"
	"strconv"
	"strings"
	"time"
)

var validEventCategories = map[string]struct{}{
	"meetup":     {},
	"concert":    {},
	"exhibition": {},
	"party":      {},
	"other":      {},
}

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
	eventID := q.Get("id")
	category := q.Get("category")
	city := q.Get("city")
	username := q.Get("user")

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

	if category != "" {
		if _, ok := validEventCategories[category]; !ok {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("category"))
			return
		}
	}

	var priceFrom *uint64
	if q.Has("price_from") {
		v, err := strconv.ParseUint(q.Get("price_from"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price_from"))
			return
		}
		priceFrom = &v
	}
	var priceTo *uint64
	if q.Has("price_to") {
		v, err := strconv.ParseUint(q.Get("price_to"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price_to"))
			return
		}
		priceTo = &v
	}
	if priceFrom != nil && priceTo != nil && *priceFrom > *priceTo {
		h.sessionHandler.WriteSessionCookie(w, r)
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
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage(dateErrField))
		return
	}

	userID := ""
	if username != "" {
		rec, err := h.userStore.FindByUsername(ctx, username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if rec == nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"events": []event.ListItem{},
				"count":  0,
			})
			return
		}
		userID = rec.ID.Hex()
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

	events, err := h.eventStore.List(ctx, event.ListFilter{
		ID:        eventID,
		Title:     title,
		Category:  category,
		City:      city,
		UserID:    userID,
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
	if shouldIncludeReactions(r) {
		if err := h.populateReactions(ctx, events); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	if shouldIncludeReviews(r) {
		if err := h.populateReviews(ctx, events); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"count":  len(events),
	})
}

func (h *HttpHandler) GetEventByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	e, err := h.eventStore.FindByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	if e == nil {
		writeJSONMessage(w, http.StatusNotFound, "Not found")
		return
	}
	if shouldIncludeReactions(r) || shouldIncludeReviews(r) {
		events := []event.ListItem{*e}
		if shouldIncludeReactions(r) {
			if err := h.populateReactions(r.Context(), events); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if shouldIncludeReviews(r) {
			if err := h.populateReviews(r.Context(), events); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		e = &events[0]
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(e)
}

func (h *HttpHandler) PatchEventByID(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPatch {
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

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}

	req := model.PatchEventRequest{}
	if rawCategory, ok := raw["category"]; ok {
		var category string
		if err := json.Unmarshal(rawCategory, &category); err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("category"))
			return
		}
		if _, valid := validEventCategories[category]; !valid {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("category"))
			return
		}
		req.Category = &category
	}
	if rawPrice, ok := raw["price"]; ok {
		var price int64
		if err := json.Unmarshal(rawPrice, &price); err != nil || price < 0 {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("price"))
			return
		}
		priceU := uint64(price)
		req.Price = &priceU
	}
	if rawCity, ok := raw["city"]; ok {
		var city string
		if err := json.Unmarshal(rawCity, &city); err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("city"))
			return
		}
		req.City = &city
	}

	updated, err := h.eventStore.PatchByIDAndOrganizer(ctx, id, userID, req)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	if !updated {
		writeJSONMessage(w, http.StatusNotFound, "Not found. Be sure that event exists and you are the organizer")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func normalizeDateRange(dateFromRaw, dateToRaw string) (string, string, string) {
	dateFrom := ""
	dateTo := ""
	if strings.TrimSpace(dateFromRaw) != "" {
		t, err := time.Parse("20060102", dateFromRaw)
		if err != nil {
			return "", "", "date_from"
		}
		dateFrom = t.Format("2006-01-02") + "T00:00:00+03:00"
	}
	if strings.TrimSpace(dateToRaw) != "" {
		t, err := time.Parse("20060102", dateToRaw)
		if err != nil {
			return "", "", "date_to"
		}
		dateTo = t.Format("2006-01-02") + "T23:59:59+03:00"
	}
	return dateFrom, dateTo, ""
}

func shouldIncludeReactions(r *http.Request) bool {
	return includeHas(r, "reactions")
}

func shouldIncludeReviews(r *http.Request) bool {
	return includeHas(r, "reviews")
}

func includeHas(r *http.Request, key string) bool {
	include := r.URL.Query().Get("include")
	for _, token := range strings.Split(include, ",") {
		if strings.TrimSpace(token) == key {
			return true
		}
	}
	return false
}

func (h *HttpHandler) populateReactions(ctx context.Context, events []event.ListItem) error {
	titles := make([]string, 0, len(events))
	for _, e := range events {
		titles = append(titles, e.Title)
	}
	byTitle, err := h.reactionService.AggregateByTitles(ctx, titles)
	if err != nil {
		return err
	}
	for i := range events {
		counts, ok := byTitle[events[i].Title]
		if !ok {
			events[i].Reactions = &event.Reactions{Likes: 0, Dislikes: 0}
			continue
		}
		events[i].Reactions = &event.Reactions{
			Likes:    counts.Likes,
			Dislikes: counts.Dislikes,
		}
	}
	return nil
}

func (h *HttpHandler) populateReviews(ctx context.Context, events []event.ListItem) error {
	titles := make([]string, 0, len(events))
	for _, e := range events {
		titles = append(titles, e.Title)
	}
	byTitle, err := h.reviewService.AggregateByTitles(ctx, titles)
	if err != nil {
		return err
	}
	for i := range events {
		agg, ok := byTitle[events[i].Title]
		if !ok {
			events[i].Reviews = &event.Reviews{Count: 0, Rating: 0}
			continue
		}
		events[i].Reviews = &event.Reviews{Count: agg.Count, Rating: agg.Rating}
	}
	return nil
}

func (h *HttpHandler) CreateEventReview(w http.ResponseWriter, r *http.Request, eventID string) {
	if r.Method != http.MethodPost {
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

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}

	rawRating, hasRating := raw["rating"]
	if !hasRating {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("rating"))
		return
	}
	var rating int
	if err := json.Unmarshal(rawRating, &rating); err != nil || rating < 1 || rating > 5 {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("rating"))
		return
	}

	rawComment, hasComment := raw["comment"]
	if !hasComment {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("comment"))
		return
	}
	var comment string
	if err := json.Unmarshal(rawComment, &comment); err != nil || len(comment) > 300 {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("comment"))
		return
	}

	e, err := h.eventStore.FindByID(ctx, eventID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if e == nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusNotFound, "Event not found")
		return
	}

	already, err := h.reviewService.ExistsByEventAndUser(ctx, eventID, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if already {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusConflict, "Already exists")
		return
	}

	id, err := h.reviewService.Create(ctx, eventID, userID, int8(rating), comment, e.Title)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func (h *HttpHandler) ListEventReviews(w http.ResponseWriter, r *http.Request, eventID string) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()
	q := r.URL.Query()

	var limit int
	var hasLimit bool
	if q.Has("limit") {
		lu, err := strconv.ParseUint(q.Get("limit"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("limit"))
			return
		}
		limit = int(lu)
		hasLimit = true
	}
	var offset int
	if q.Has("offset") {
		ou, err := strconv.ParseUint(q.Get("offset"), 10, 64)
		if err != nil {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("offset"))
			return
		}
		offset = int(ou)
	}

	if hasLimit && limit == 0 {
		h.sessionHandler.WriteSessionCookie(w, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"reviews": []struct{}{}, "count": 0})
		return
	}

	all, err := h.reviewService.ListByEventID(ctx, eventID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Apply offset/limit in memory (Cassandra has no OFFSET)
	if offset >= len(all) {
		h.sessionHandler.WriteSessionCookie(w, r)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"reviews": []struct{}{}, "count": 0})
		return
	}
	all = all[offset:]
	if hasLimit && limit < len(all) {
		all = all[:limit]
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"reviews": all, "count": len(all)})
}

func (h *HttpHandler) PatchEventReview(w http.ResponseWriter, r *http.Request, eventID, reviewID string) {
	if r.Method != http.MethodPatch {
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

	e, err := h.eventStore.FindByID(ctx, eventID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if e == nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusNotFound, "Event not found")
		return
	}

	rev, err := h.reviewService.FindByEventAndID(ctx, eventID, reviewID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if rev == nil || rev.CreatedBy != userID {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusNotFound, "Event not found")
		return
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("body"))
		return
	}

	var newRating *int8
	if rawRating, ok := raw["rating"]; ok {
		var v int
		if err := json.Unmarshal(rawRating, &v); err != nil || v < 1 || v > 5 {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("rating"))
			return
		}
		r8 := int8(v)
		newRating = &r8
	}
	var newComment *string
	if rawComment, ok := raw["comment"]; ok {
		var v string
		if err := json.Unmarshal(rawComment, &v); err != nil || len(v) > 300 {
			h.sessionHandler.WriteSessionCookie(w, r)
			writeJSONMessage(w, http.StatusBadRequest, invalidFieldMessage("comment"))
			return
		}
		newComment = &v
	}

	if err := h.reviewService.Update(ctx, eventID, rev.CreatedBy, newRating, newComment, e.Title); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	h.sessionHandler.WriteSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *HttpHandler) PutEventLike(w http.ResponseWriter, r *http.Request, id string) {
	h.putEventReaction(w, r, id, true)
}

func (h *HttpHandler) PutEventDislike(w http.ResponseWriter, r *http.Request, id string) {
	h.putEventReaction(w, r, id, false)
}

func (h *HttpHandler) putEventReaction(w http.ResponseWriter, r *http.Request, id string, isLike bool) {
	if r.Method != http.MethodPost {
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
		expireSessionCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	e, err := h.eventStore.FindByID(ctx, id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if e == nil {
		h.sessionHandler.WriteSessionCookie(w, r)
		writeJSONMessage(w, http.StatusNotFound, "Event not found")
		return
	}
	if isLike {
		err = h.reactionService.PutLike(ctx, id, userID, e.Title)
	} else {
		err = h.reactionService.PutDislike(ctx, id, userID, e.Title)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.sessionHandler.WriteSessionCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}
