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

func (s *Service) PutLike(ctx context.Context, eventID string, userID string) error {
	return s.store.Upsert(ctx, eventID, userID, 1)
}

func (s *Service) PutDislike(ctx context.Context, eventID string, userID string) error {
	return s.store.Upsert(ctx, eventID, userID, -1)
}

func (s *Service) InvalidateTitle(ctx context.Context, title string) error {
	return s.cache.Delete(ctx, title)
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
		if total.Likes > 0 || total.Dislikes > 0 {
			if err := s.cache.Set(ctx, title, total, s.likeTTL); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

func (s *Service) WarmEventCache(ctx context.Context, eventID string, title string) error {
	perEvent, err := s.store.CountByEventIDs(ctx, []string{eventID})
	if err != nil {
		return err
	}
	counters := perEvent[eventID]
	if counters.Likes == 0 && counters.Dislikes == 0 {
		return nil
	}
	if strings.TrimSpace(title) != "" {
		if err := s.cache.Set(ctx, title, counters, s.likeTTL); err != nil {
			return err
		}
	}
	return s.cache.Set(ctx, eventID, counters, s.likeTTL)
}
