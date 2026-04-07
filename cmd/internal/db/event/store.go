package event

import (
	"context"
	"errors"
	"nosql-labs/cmd/internal/model"
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
	ID        string
	Title     string
	Category  string
	City      string
	UserID    string
	PriceFrom *uint64
	PriceTo   *uint64
	DateFrom  string
	DateTo    string
	Limit     int64
	Offset    int64
}

func (s *EventStore) List(ctx context.Context, f ListFilter) ([]ListItem, error) {
	filter := bson.M{}
	if strings.TrimSpace(f.ID) != "" {
		oid, err := bson.ObjectIDFromHex(f.ID)
		if err != nil {
			return []ListItem{}, nil
		}
		filter["_id"] = oid
	}
	if strings.TrimSpace(f.Title) != "" {
		filter["title"] = bson.M{"$regex": f.Title, "$options": "i"}
	}
	if strings.TrimSpace(f.Category) != "" {
		filter["category"] = f.Category
	}
	if strings.TrimSpace(f.City) != "" {
		filter["location.city"] = f.City
	}
	if strings.TrimSpace(f.UserID) != "" {
		filter["created_by"] = f.UserID
	}
	if f.PriceFrom != nil || f.PriceTo != nil {
		priceFilter := bson.M{}
		if f.PriceFrom != nil {
			priceFilter["$gte"] = *f.PriceFrom
		}
		if f.PriceTo != nil {
			priceFilter["$lte"] = *f.PriceTo
		}
		filter["price"] = priceFilter
	}
	if strings.TrimSpace(f.DateFrom) != "" || strings.TrimSpace(f.DateTo) != "" {
		dateFilter := bson.M{}
		if strings.TrimSpace(f.DateFrom) != "" {
			dateFilter["$gte"] = f.DateFrom
		}
		if strings.TrimSpace(f.DateTo) != "" {
			dateFilter["$lte"] = f.DateTo
		}
		filter["started_at"] = dateFilter
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
			Category:    raw.Category,
			Price:       raw.Price,
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

func (s *EventStore) FindByID(ctx context.Context, idHex string) (*ListItem, error) {
	oid, err := bson.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, nil
	}
	var raw EventRecord
	err = s.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&raw)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &ListItem{
		ID:          raw.ID.Hex(),
		Title:       raw.Title,
		Category:    raw.Category,
		Price:       raw.Price,
		Description: raw.Description,
		Location:    raw.Location,
		CreatedAt:   raw.CreatedAt,
		CreatedBy:   raw.CreatedBy,
		StartedAt:   raw.StartedAt,
		FinishedAt:  raw.FinishedAt,
	}, nil
}

func (s *EventStore) PatchByIDAndOrganizer(ctx context.Context, idHex, organizerID string, req model.PatchEventRequest) (bool, error) {
	oid, err := bson.ObjectIDFromHex(idHex)
	if err != nil {
		return false, nil
	}
	setDoc := bson.M{}
	unsetDoc := bson.M{}
	if req.Category != nil {
		setDoc["category"] = *req.Category
	}
	if req.Price != nil {
		setDoc["price"] = *req.Price
	}
	if req.City != nil {
		if *req.City == "" {
			unsetDoc["location.city"] = ""
		} else {
			setDoc["location.city"] = *req.City
		}
	}
	update := bson.M{}
	if len(setDoc) > 0 {
		update["$set"] = setDoc
	}
	if len(unsetDoc) > 0 {
		update["$unset"] = unsetDoc
	}
	if len(update) == 0 {
		exists, err := s.coll.CountDocuments(ctx, bson.M{"_id": oid, "created_by": organizerID})
		if err != nil {
			return false, err
		}
		return exists > 0, nil
	}

	res, err := s.coll.UpdateOne(ctx, bson.M{"_id": oid, "created_by": organizerID}, update)
	if err != nil {
		return false, err
	}
	return res.MatchedCount > 0, nil
}
