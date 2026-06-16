package graph

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Store struct {
	driver neo4j.DriverWithContext
}

func NewStore(driver neo4j.DriverWithContext) *Store {
	return &Store{driver: driver}
}

func (s *Store) EnsureUser(ctx context.Context, userID string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, "MERGE (:User {id: $id})", map[string]any{"id": userID})
		return nil, err
	})
	return err
}

func (s *Store) EnsureEvent(ctx context.Context, eventID, title string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			"MERGE (e:Event {id: $id}) SET e.title = $title",
			map[string]any{"id": eventID, "title": title},
		)
		return nil, err
	})
	return err
}

func (s *Store) AddLike(ctx context.Context, userID, eventID string) error {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx,
			`MERGE (u:User {id: $userID})
			 MERGE (e:Event {id: $eventID})
			 MERGE (u)-[:LIKED]->(e)`,
			map[string]any{"userID": userID, "eventID": eventID},
		)
		return nil, err
	})
	return err
}

type RecommendedEvent struct {
	EventID string
	Score   int64
}

func (s *Store) GetRecommendations(ctx context.Context, userID string) ([]RecommendedEvent, error) {
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cursor, err := tx.Run(ctx,
			`MATCH (me:User {id: $userID})-[:LIKED]->(e:Event)<-[:LIKED]-(other:User)-[:LIKED]->(rec:Event)
			 WHERE NOT (me)-[:LIKED]->(rec)
			 RETURN rec.id AS eventID, count(DISTINCT other) AS score
			 ORDER BY score DESC`,
			map[string]any{"userID": userID},
		)
		if err != nil {
			return nil, err
		}
		var recs []RecommendedEvent
		for cursor.Next(ctx) {
			record := cursor.Record()
			eventID, _ := record.Get("eventID")
			score, _ := record.Get("score")
			recs = append(recs, RecommendedEvent{
				EventID: eventID.(string),
				Score:   score.(int64),
			})
		}
		return recs, cursor.Err()
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]RecommendedEvent), nil
}
