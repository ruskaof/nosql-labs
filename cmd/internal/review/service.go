package review

import (
	"context"
	"math"
	"strings"
	"time"

	"nosql-labs/cmd/internal/db/event"

	"github.com/gocql/gocql"
)

type EventStore interface {
	ListByTitle(ctx context.Context, title string) ([]event.ListItem, error)
}

type Service struct {
	store      Store
	cache      *Cache
	reviewsTTL time.Duration
	eventStore EventStore
}

func NewService(store Store, cache *Cache, reviewsTTL time.Duration, eventStore EventStore) *Service {
	return &Service{store: store, cache: cache, reviewsTTL: reviewsTTL, eventStore: eventStore}
}

func (s *Service) AggregateByTitles(ctx context.Context, titles []string) (map[string]Aggregates, error) {
	out := make(map[string]Aggregates, len(titles))
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
		agg, err := s.computeAndCacheByTitle(ctx, title)
		if err != nil {
			return nil, err
		}
		out[title] = agg
	}
	return out, nil
}

func (s *Service) Create(ctx context.Context, eventID, userID string, rating int8, comment string, eventTitle string) (string, error) {
	id := gocql.TimeUUID().String()
	now := time.Now()
	if err := s.store.Create(ctx, id, eventID, userID, rating, comment, now); err != nil {
		return "", err
	}
	if err := s.RefreshTitleCache(ctx, eventTitle); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) ExistsByEventAndUser(ctx context.Context, eventID, userID string) (bool, error) {
	return s.store.ExistsByEventAndUser(ctx, eventID, userID)
}

func (s *Service) FindByEventAndID(ctx context.Context, eventID, id string) (*Review, error) {
	return s.store.FindByEventAndID(ctx, eventID, id)
}

func (s *Service) ListByEventID(ctx context.Context, eventID string) ([]Review, error) {
	return s.store.ListByEventID(ctx, eventID)
}

func (s *Service) Update(ctx context.Context, eventID, createdBy string, rating *int8, comment *string, eventTitle string) error {
	if err := s.store.Update(ctx, eventID, createdBy, rating, comment, time.Now()); err != nil {
		return err
	}
	return s.RefreshTitleCache(ctx, eventTitle)
}

func (s *Service) RefreshTitleCache(ctx context.Context, title string) error {
	_, err := s.computeAndCacheByTitle(ctx, title)
	return err
}

func (s *Service) computeAndCacheByTitle(ctx context.Context, title string) (Aggregates, error) {
	eventsByTitle, err := s.eventStore.ListByTitle(ctx, title)
	if err != nil {
		return Aggregates{}, err
	}
	eventIDs := make([]string, 0, len(eventsByTitle))
	for _, e := range eventsByTitle {
		eventIDs = append(eventIDs, e.ID)
	}
	ratings, err := s.store.GetRatingsByEventIDs(ctx, eventIDs)
	if err != nil {
		return Aggregates{}, err
	}
	agg := Aggregates{}
	if len(ratings) > 0 {
		sum := 0
		for _, r := range ratings {
			sum += int(r)
		}
		agg.Count = len(ratings)
		agg.Rating = math.Round(float64(sum)/float64(len(ratings))*10) / 10
	}
	if err := s.cache.Set(ctx, title, agg, s.reviewsTTL); err != nil {
		return Aggregates{}, err
	}
	return agg, nil
}
