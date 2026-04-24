package reaction

import "context"

type Counters struct {
	Likes    int `json:"likes"`
	Dislikes int `json:"dislikes"`
}

type Store interface {
	Upsert(ctx context.Context, eventID string, userID string, likeValue int8) error
	CountByEventIDs(ctx context.Context, eventIDs []string) (map[string]Counters, error)
	InitSchema(ctx context.Context) error
}
