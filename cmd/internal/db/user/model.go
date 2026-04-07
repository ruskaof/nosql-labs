package user

import "go.mongodb.org/mongo-driver/v2/bson"

type UserRecord struct {
	ID           bson.ObjectID `bson:"_id"`
	FullName     string        `bson:"full_name"`
	Username     string        `bson:"username"`
	PasswordHash string        `bson:"password_hash"`
}

type PublicUser struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
}
