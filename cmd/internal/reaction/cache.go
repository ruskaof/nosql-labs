package reaction

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
	return "events:" + hex.EncodeToString(sum[:]) + ":reactions"
}

func (c *Cache) plainKey(token string) string {
	return "events:" + token + ":reactions"
}

func (c *Cache) Get(ctx context.Context, title string) (Counters, bool, error) {
	raw, err := c.client.Get(ctx, c.key(title)).Result()
	if err == redis.Nil {
		raw, err = c.client.Get(ctx, c.plainKey(title)).Result()
		if err == redis.Nil {
			return Counters{}, false, nil
		}
		if err != nil {
			return Counters{}, false, err
		}
	}
	if err != nil {
		return Counters{}, false, err
	}
	var counters Counters
	if err := json.Unmarshal([]byte(raw), &counters); err != nil {
		return Counters{}, false, err
	}
	return counters, true, nil
}

func (c *Cache) Set(ctx context.Context, title string, counters Counters, ttl time.Duration) error {
	payload, err := json.Marshal(counters)
	if err != nil {
		return err
	}
	pipe := c.client.TxPipeline()
	pipe.Set(ctx, c.key(title), payload, ttl)
	pipe.Set(ctx, c.plainKey(title), payload, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *Cache) Delete(ctx context.Context, title string) error {
	return c.client.Del(ctx, c.key(title), c.plainKey(title)).Err()
}
