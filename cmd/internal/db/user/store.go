package user

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
