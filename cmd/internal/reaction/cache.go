package reaction

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
	return "event:" + hex.EncodeToString(sum[:]) + ":reactions"
}

func (c *Cache) Get(ctx context.Context, title string) (Counters, bool, error) {
	values, err := c.client.HGetAll(ctx, c.key(title)).Result()
	if err != nil {
		return Counters{}, false, err
	}
	if len(values) == 0 {
		return Counters{}, false, nil
	}
	likesRaw, okLikes := values["likes"]
	dislikesRaw, okDislikes := values["dislikes"]
	if !okLikes || !okDislikes {
		return Counters{}, false, nil
	}
	likes, err := strconv.Atoi(likesRaw)
	if err != nil {
		return Counters{}, false, err
	}
	dislikes, err := strconv.Atoi(dislikesRaw)
	if err != nil {
		return Counters{}, false, err
	}
	return Counters{Likes: likes, Dislikes: dislikes}, true, nil
}

func (c *Cache) Set(ctx context.Context, title string, counters Counters, ttl time.Duration) error {
	key := c.key(title)
	pipe := c.client.TxPipeline()
	pipe.HSet(ctx, key, map[string]interface{}{
		"likes":    counters.Likes,
		"dislikes": counters.Dislikes,
	})
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Cache) Delete(ctx context.Context, title string) error {
	return c.client.Del(ctx, c.key(title)).Err()
}
