package event

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrExists = errors.New("event already exists")

type EventStore struct {
	coll *mongo.Collection
}

func NewStore(db *mongo.Database) *EventStore {
	return &EventStore{coll: db.Collection("events")}
}

func (s *EventStore) Create(ctx context.Context, title, description, address, createdByHex, startedAt, finishedAt string) (bson.ObjectID, error) {
	created := time.Now().Format(time.RFC3339)
	doc := bson.M{
		"title":       title,
		"description": description,
		"location":    bson.M{"address": address},
		"created_at":  created,
		"created_by":  createdByHex,
		"started_at":  startedAt,
		"finished_at": finishedAt,
	}
	res, err := s.coll.InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return bson.ObjectID{}, ErrExists
		}
		return bson.ObjectID{}, err
	}
	oid, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return bson.ObjectID{}, errors.New("unexpected inserted id type")
	}
	return oid, nil
}

type ListFilter struct {
	Title  string
	Limit  int64
	Offset int64
}

func (s *EventStore) List(ctx context.Context, f ListFilter) ([]ListItem, error) {
	filter := bson.M{}
	if strings.TrimSpace(f.Title) != "" {
		filter["title"] = bson.M{"$regex": f.Title, "$options": "i"}
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(f.Offset)
	if f.Limit > 0 {
		opts.SetLimit(f.Limit)
	}
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var out []ListItem
	for cursor.Next(ctx) {
		var raw EventRecord
		if err := cursor.Decode(&raw); err != nil {
			return nil, err
		}
		out = append(out, ListItem{
			ID:          raw.ID.Hex(),
			Title:       raw.Title,
			Description: raw.Description,
			Location:    raw.Location,
			CreatedAt:   raw.CreatedAt,
			CreatedBy:   raw.CreatedBy,
			StartedAt:   raw.StartedAt,
			FinishedAt:  raw.FinishedAt,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []ListItem{}
	}
	return out, nil
}
