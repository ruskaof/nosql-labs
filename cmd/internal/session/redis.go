package session

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "sid:"

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) key(id string) string {
	return keyPrefix + id
}

func (s *RedisStore) Exists(ctx context.Context, sessionID string) (bool, error) {
	n, err := s.client.Exists(ctx, s.key(sessionID)).Result()
	if err != nil {
	    if errors.Is(err, redis.Nil) {
	        return false, nil
	    }
		return false, err
	}
	return n > 0, nil
}

func (s *RedisStore) Save(ctx context.Context, sessionID string, ttl time.Duration) error {
	now := time.Now().Format(time.RFC3339)
	key := s.key(sessionID)
	if err := s.client.HSetEX(ctx, key, "created_at", now, "updated_at", now).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, key, ttl).Err()
}

func (s *RedisStore) Update(ctx context.Context, sessionID string, ttl time.Duration) error {
	key := s.key(sessionID)
	if err := s.client.HSetEX(ctx, key, "updated_at", time.Now().Format(time.RFC3339)).Err(); err != nil {
		return err
	}
	return s.client.Expire(ctx, key, ttl).Err()
}
