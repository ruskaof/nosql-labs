package review

import (
	"context"
	"time"
)

type Review struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Comment   string    `json:"comment"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
	Rating    int8      `json:"rating"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Aggregates struct {
	Count  int     `json:"count"`
	Rating float64 `json:"rating"`
}

type Store interface {
	Create(ctx context.Context, id, eventID, userID string, rating int8, comment string, now time.Time) error
	ExistsByEventAndUser(ctx context.Context, eventID, userID string) (bool, error)
	FindByEventAndID(ctx context.Context, eventID, id string) (*Review, error)
	Update(ctx context.Context, eventID, createdBy string, rating *int8, comment *string, updatedAt time.Time) error
	ListByEventID(ctx context.Context, eventID string) ([]Review, error)
	GetRatingsByEventIDs(ctx context.Context, eventIDs []string) ([]int8, error)
}
