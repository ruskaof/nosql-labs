package review

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strconv"
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
	values, err := c.client.HGetAll(ctx, c.key(title)).Result()
	if err != nil {
		return Aggregates{}, false, err
	}
	if len(values) == 0 {
		return Aggregates{}, false, nil
	}
	countRaw, okCount := values["count"]
	ratingRaw, okRating := values["rating"]
	if !okCount && !okRating {
		return Aggregates{}, false, nil
	}
	count, err := strconv.Atoi(countRaw)
	if err != nil {
		return Aggregates{}, false, err
	}
	rating, err := strconv.ParseFloat(ratingRaw, 64)
	if err != nil {
		return Aggregates{}, false, err
	}
	return Aggregates{Count: count, Rating: rating}, true, nil
}

func (c *Cache) Set(ctx context.Context, title string, agg Aggregates, ttl time.Duration) error {
	key := c.key(title)
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"count":  agg.Count,
		"rating": strconv.FormatFloat(agg.Rating, 'f', 1, 64),
	})
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}
