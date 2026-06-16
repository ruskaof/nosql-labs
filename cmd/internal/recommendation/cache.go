package recommendation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"nosql-labs/cmd/internal/db/event"
)

type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) key(userID string) string {
	return "user:" + userID + ":recomms"
}

func (c *Cache) Get(ctx context.Context, userID string) ([]event.ListItem, bool, error) {
	val, err := c.client.HGet(ctx, c.key(userID), "events").Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var events []event.ListItem
	if err := json.Unmarshal([]byte(val), &events); err != nil {
		return nil, false, err
	}
	return events, true, nil
}

func (c *Cache) Set(ctx context.Context, userID string, events []event.ListItem, ttl time.Duration) error {
	data, err := json.Marshal(events)
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, c.key(userID), "events", string(data))
	pipe.Expire(ctx, c.key(userID), ttl)
	_, err = pipe.Exec(ctx)
	return err
}
