package recommendation

import (
	"context"
	"sort"
	"time"

	"nosql-labs/cmd/internal/db/event"
	"nosql-labs/cmd/internal/graph"
)

type EventStore interface {
	FindManyByIDs(ctx context.Context, ids []string) ([]event.ListItem, error)
}

type Service struct {
	graphStore *graph.Store
	eventStore EventStore
	cache      *Cache
	ttl        time.Duration
}

func NewService(graphStore *graph.Store, eventStore EventStore, cache *Cache, ttl time.Duration) *Service {
	return &Service{
		graphStore: graphStore,
		eventStore: eventStore,
		cache:      cache,
		ttl:        ttl,
	}
}

func (s *Service) GetRecommendations(ctx context.Context, userID string) ([]event.ListItem, error) {
	cached, ok, err := s.cache.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if ok {
		return cached, nil
	}

	recs, err := s.graphStore.GetRecommendations(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(recs) == 0 {
		if err := s.cache.Set(ctx, userID, []event.ListItem{}, s.ttl); err != nil {
			return nil, err
		}
		return []event.ListItem{}, nil
	}

	ids := make([]string, len(recs))
	scoreByID := make(map[string]int64, len(recs))
	for i, r := range recs {
		ids[i] = r.EventID
		scoreByID[r.EventID] = r.Score
	}

	events, err := s.eventStore.FindManyByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	type titleGroup struct {
		ev    event.ListItem
		score int64
	}
	byTitle := make(map[string]titleGroup)
	for _, e := range events {
		score := scoreByID[e.ID]
		existing, exists := byTitle[e.Title]
		if !exists {
			byTitle[e.Title] = titleGroup{ev: e, score: score}
			continue
		}
		maxScore := existing.score
		if score > maxScore {
			maxScore = score
		}
		if e.StartedAt < existing.ev.StartedAt {
			byTitle[e.Title] = titleGroup{ev: e, score: maxScore}
		} else {
			byTitle[e.Title] = titleGroup{ev: existing.ev, score: maxScore}
		}
	}

	groups := make([]titleGroup, 0, len(byTitle))
	for _, g := range byTitle {
		groups = append(groups, g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].score > groups[j].score
	})

	out := make([]event.ListItem, len(groups))
	for i, g := range groups {
		out[i] = g.ev
	}

	if err := s.cache.Set(ctx, userID, out, s.ttl); err != nil {
		return nil, err
	}
	return out, nil
}
