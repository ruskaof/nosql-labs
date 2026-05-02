package reaction

import (
	"context"
	"time"

	"github.com/gocql/gocql"
)

type CassandraStore struct {
	session *gocql.Session
}

func NewCassandraStore(session *gocql.Session, _ string) *CassandraStore {
	return &CassandraStore{session: session}
}

func (s *CassandraStore) InitSchema(ctx context.Context) error {
	createTable := `CREATE TABLE IF NOT EXISTS event_reactions (
		event_id text,
		created_by text,
		like_value tinyint,
		created_at timestamp,
		PRIMARY KEY ((event_id), created_by)
	)`
	if err := s.session.Query(createTable).WithContext(ctx).Exec(); err != nil {
		return err
	}
	createLikeValueIndex := `CREATE INDEX IF NOT EXISTS event_reactions_like_value_idx ON event_reactions (like_value)`
	if err := s.session.Query(createLikeValueIndex).WithContext(ctx).Exec(); err != nil {
		return err
	}
	createCreatedByIndex := `CREATE INDEX IF NOT EXISTS event_reactions_created_by_idx ON event_reactions (created_by)`
	return s.session.Query(createCreatedByIndex).WithContext(ctx).Exec()
}

func (s *CassandraStore) Upsert(ctx context.Context, eventID string, userID string, likeValue int8) error {
	query := `INSERT INTO event_reactions (event_id, created_by, like_value, created_at) VALUES (?, ?, ?, ?)`
	return s.session.Query(query, eventID, userID, likeValue, time.Now().UTC()).WithContext(ctx).Exec()
}

func (s *CassandraStore) CountByEventIDs(ctx context.Context, eventIDs []string) (map[string]Counters, error) {
	out := make(map[string]Counters, len(eventIDs))
	for _, eventID := range eventIDs {
		iter := s.session.Query(`SELECT like_value FROM event_reactions WHERE event_id = ?`, eventID).WithContext(ctx).Iter()
		var likeValue int8
		c := Counters{}
		for iter.Scan(&likeValue) {
			if likeValue == 1 {
				c.Likes++
			} else if likeValue == -1 {
				c.Dislikes++
			}
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
		out[eventID] = c
	}
	return out, nil
}
