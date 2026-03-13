package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

type Store interface {
	Exists(ctx context.Context, sessionID string) (bool, error)
	Save(ctx context.Context, sessionID string, ttl time.Duration) error
	Update(ctx context.Context, sessionID string, ttl time.Duration) error
}

func GenerateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
