package review

import (
	"context"
	"time"

	"github.com/gocql/gocql"
)

type CassandraStore struct {
	session *gocql.Session
}

func NewCassandraStore(session *gocql.Session) *CassandraStore {
	return &CassandraStore{session: session}
}

func (s *CassandraStore) Create(ctx context.Context, id, eventID, userID string, rating int8, comment string, now time.Time) error {
	uid, err := gocql.ParseUUID(id)
	if err != nil {
		return err
	}
	return s.session.Query(
		`INSERT INTO event_reviews (event_id, created_by, id, rating, comment, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventID, userID, uid, rating, comment, now.UTC(), now.UTC(),
	).WithContext(ctx).Exec()
}

func (s *CassandraStore) ExistsByEventAndUser(ctx context.Context, eventID, userID string) (bool, error) {
	var count int
	err := s.session.Query(
		`SELECT COUNT(*) FROM event_reviews WHERE event_id = ? AND created_by = ?`,
		eventID, userID,
	).WithContext(ctx).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *CassandraStore) FindByEventAndID(ctx context.Context, eventID, id string) (*Review, error) {
	uid, err := gocql.ParseUUID(id)
	if err != nil {
		return nil, nil
	}
	var r Review
	var cassID gocql.UUID
	var createdAt, updatedAt time.Time
	err = s.session.Query(
		`SELECT id, event_id, created_by, rating, comment, created_at, updated_at FROM event_reviews WHERE event_id = ? AND id = ? ALLOW FILTERING`,
		eventID, uid,
	).WithContext(ctx).Scan(&cassID, &r.EventID, &r.CreatedBy, &r.Rating, &r.Comment, &createdAt, &updatedAt)
	if err == gocql.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r.ID = cassID.String()
	r.CreatedAt = createdAt
	r.UpdatedAt = updatedAt
	return &r, nil
}

func (s *CassandraStore) Update(ctx context.Context, eventID, createdBy string, rating *int8, comment *string, updatedAt time.Time) error {
	switch {
	case rating != nil && comment != nil:
		return s.session.Query(
			`UPDATE event_reviews SET rating = ?, comment = ?, updated_at = ? WHERE event_id = ? AND created_by = ?`,
			*rating, *comment, updatedAt.UTC(), eventID, createdBy,
		).WithContext(ctx).Exec()
	case rating != nil:
		return s.session.Query(
			`UPDATE event_reviews SET rating = ?, updated_at = ? WHERE event_id = ? AND created_by = ?`,
			*rating, updatedAt.UTC(), eventID, createdBy,
		).WithContext(ctx).Exec()
	case comment != nil:
		return s.session.Query(
			`UPDATE event_reviews SET comment = ?, updated_at = ? WHERE event_id = ? AND created_by = ?`,
			*comment, updatedAt.UTC(), eventID, createdBy,
		).WithContext(ctx).Exec()
	default:
		return s.session.Query(
			`UPDATE event_reviews SET updated_at = ? WHERE event_id = ? AND created_by = ?`,
			updatedAt.UTC(), eventID, createdBy,
		).WithContext(ctx).Exec()
	}
}

func (s *CassandraStore) ListByEventID(ctx context.Context, eventID string) ([]Review, error) {
	iter := s.session.Query(
		`SELECT id, event_id, created_by, rating, comment, created_at, updated_at FROM event_reviews WHERE event_id = ?`,
		eventID,
	).WithContext(ctx).Iter()

	var reviews []Review
	var cassID gocql.UUID
	var evID, createdBy, comment string
	var rating int8
	var createdAt, updatedAt time.Time
	for iter.Scan(&cassID, &evID, &createdBy, &rating, &comment, &createdAt, &updatedAt) {
		reviews = append(reviews, Review{
			ID:        cassID.String(),
			EventID:   evID,
			CreatedBy: createdBy,
			Rating:    rating,
			Comment:   comment,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}
	if err := iter.Close(); err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []Review{}
	}
	return reviews, nil
}

func (s *CassandraStore) GetRatingsByEventIDs(ctx context.Context, eventIDs []string) ([]int8, error) {
	var ratings []int8
	for _, eventID := range eventIDs {
		iter := s.session.Query(
			`SELECT rating FROM event_reviews WHERE event_id = ?`, eventID,
		).WithContext(ctx).Iter()
		var rating int8
		for iter.Scan(&rating) {
			ratings = append(ratings, rating)
		}
		if err := iter.Close(); err != nil {
			return nil, err
		}
	}
	return ratings, nil
}
