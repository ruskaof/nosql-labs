package user

import (
	"context"
	"errors"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var ErrExists = errors.New("user already exists")

type UserStore struct {
	coll *mongo.Collection
}

func NewStore(db *mongo.Database) *UserStore {
	return &UserStore{coll: db.Collection("users")}
}

func (s *UserStore) Create(ctx context.Context, fullName, username, passwordHash string) (bson.ObjectID, error) {
	doc := bson.M{
		"full_name":     fullName,
		"username":      username,
		"password_hash": passwordHash,
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

func (s *UserStore) FindByUsername(ctx context.Context, username string) (*UserRecord, error) {
	var rec UserRecord
	err := s.coll.FindOne(ctx, bson.M{"username": username}).Decode(&rec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (s *UserStore) FindByID(ctx context.Context, idHex string) (*PublicUser, error) {
	oid, err := bson.ObjectIDFromHex(idHex)
	if err != nil {
		return nil, nil
	}
	var rec UserRecord
	err = s.coll.FindOne(ctx, bson.M{"_id": oid}).Decode(&rec)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &PublicUser{
		ID:       rec.ID.Hex(),
		FullName: rec.FullName,
		Username: rec.Username,
	}, nil
}

type ListFilter struct {
	ID     string
	Name   string
	Limit  int64
	Offset int64
}

func (s *UserStore) List(ctx context.Context, f ListFilter) ([]PublicUser, error) {
	filter := bson.M{}
	if strings.TrimSpace(f.ID) != "" {
		oid, err := bson.ObjectIDFromHex(f.ID)
		if err != nil {
			return []PublicUser{}, nil
		}
		filter["_id"] = oid
	}
	if strings.TrimSpace(f.Name) != "" {
		filter["full_name"] = bson.M{"$regex": f.Name, "$options": "i"}
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "full_name", Value: 1}}).
		SetSkip(f.Offset)
	if f.Limit > 0 {
		opts.SetLimit(f.Limit)
	}
	cursor, err := s.coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	users := make([]PublicUser, 0)
	for cursor.Next(ctx) {
		var rec UserRecord
		if err := cursor.Decode(&rec); err != nil {
			return nil, err
		}
		users = append(users, PublicUser{
			ID:       rec.ID.Hex(),
			FullName: rec.FullName,
			Username: rec.Username,
		})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
