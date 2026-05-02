package reaction

import (
	"context"
	"strings"
	"time"

	"nosql-labs/cmd/internal/db/event"
)

type EventStore interface {
	ListByTitle(ctx context.Context, title string) ([]event.ListItem, error)
}

type Service struct {
	store      Store
	cache      *Cache
	likeTTL    time.Duration
	eventStore EventStore
}

func NewService(store Store, cache *Cache, likeTTL time.Duration, eventStore EventStore) *Service {
	return &Service{store: store, cache: cache, likeTTL: likeTTL, eventStore: eventStore}
}

func (s *Service) PutLike(ctx context.Context, eventID string, userID string, title string) error {
	if err := s.store.Upsert(ctx, eventID, userID, 1); err != nil {
		return err
	}
	_, err := s.refreshTitleCache(ctx, title)
	return err
}

func (s *Service) PutDislike(ctx context.Context, eventID string, userID string, title string) error {
	if err := s.store.Upsert(ctx, eventID, userID, -1); err != nil {
		return err
	}
	_, err := s.refreshTitleCache(ctx, title)
	return err
}

func (s *Service) AggregateByTitles(ctx context.Context, titles []string) (map[string]Counters, error) {
	out := make(map[string]Counters, len(titles))
	seen := map[string]struct{}{}
	for _, title := range titles {
		title = strings.TrimSpace(title)
		if title == "" {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		cached, ok, err := s.cache.Get(ctx, title)
		if err != nil {
			return nil, err
		}
		if ok {
			out[title] = cached
			continue
		}
		eventsByTitle, err := s.eventStore.ListByTitle(ctx, title)
		if err != nil {
			return nil, err
		}
		eventIDs := make([]string, 0, len(eventsByTitle))
		for _, e := range eventsByTitle {
			eventIDs = append(eventIDs, e.ID)
		}
		perEvent, err := s.store.CountByEventIDs(ctx, eventIDs)
		if err != nil {
			return nil, err
		}
		total := Counters{}
		for _, c := range perEvent {
			total.Likes += c.Likes
			total.Dislikes += c.Dislikes
		}
		out[title] = total
		if err := s.cache.Set(ctx, title, total, s.likeTTL); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) refreshTitleCache(ctx context.Context, title string) (Counters, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Counters{}, nil
	}
	eventsByTitle, err := s.eventStore.ListByTitle(ctx, title)
	if err != nil {
		return Counters{}, err
	}
	eventIDs := make([]string, 0, len(eventsByTitle))
	for _, e := range eventsByTitle {
		eventIDs = append(eventIDs, e.ID)
	}
	perEvent, err := s.store.CountByEventIDs(ctx, eventIDs)
	if err != nil {
		return Counters{}, err
	}
	total := Counters{}
	for _, c := range perEvent {
		total.Likes += c.Likes
		total.Dislikes += c.Dislikes
	}
	if err := s.cache.Set(ctx, title, total, s.likeTTL); err != nil {
		return Counters{}, err
	}
	return total, nil
}
