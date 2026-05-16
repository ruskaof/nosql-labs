package review

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache struct {
	client *redis.Client
}

func NewCache(client *redis.Client) *Cache {
	return &Cache{client: client}
}

func (c *Cache) key(title string) string {
	sum := md5.Sum([]byte(title))
	return "event:" + hex.EncodeToString(sum[:]) + ":reviews"
}

func (c *Cache) Get(ctx context.Context, title string) (Aggregates, bool, error) {
	val, err := c.client.Get(ctx, c.key(title)).Result()
	if err == redis.Nil {
		return Aggregates{}, false, nil
	}
	if err != nil {
		return Aggregates{}, false, err
	}
	var agg Aggregates
	if err := json.Unmarshal([]byte(val), &agg); err != nil {
		return Aggregates{}, false, err
	}
	return agg, true, nil
}

func (c *Cache) Set(ctx context.Context, title string, agg Aggregates, ttl time.Duration) error {
	data, err := json.Marshal(agg)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(title), data, ttl).Err()
}
