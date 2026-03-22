package event

import "go.mongodb.org/mongo-driver/v2/bson"

type locationDoc struct {
	Address string `bson:"address" json:"address"`
}

type ListItem struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description,omitempty"`
	Location    locationDoc `json:"location"`
	CreatedAt   string      `json:"created_at"`
	CreatedBy   string      `json:"created_by"`
	StartedAt   string      `json:"started_at"`
	FinishedAt  string      `json:"finished_at"`
}

type EventRecord struct {
	ID          bson.ObjectID `bson:"_id"`
	Title       string        `bson:"title"`
	Description string        `bson:"description"`
	Location    locationDoc   `bson:"location"`
	CreatedAt   string        `bson:"created_at"`
	CreatedBy   string        `bson:"created_by"`
	StartedAt   string        `bson:"started_at"`
	FinishedAt  string        `bson:"finished_at"`
}