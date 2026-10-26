package cache

import (
	"context"
	"time"
)

const SessionCacheTTL = 3 * time.Minute

func SessionKey(tokenHash string) string {
	return "auth:session:" + tokenHash
}

// CacheSession marks a token hash as valid in Redis.
func CacheSession(tokenHash string) {
	if RDB == nil {
		return
	}
	RDB.Set(context.Background(), SessionKey(tokenHash), "1", SessionCacheTTL)
}

// InvalidateSession removes a token hash from the Redis session cache immediately.
func InvalidateSession(tokenHash string) {
	if RDB == nil {
		return
	}
	RDB.Del(context.Background(), SessionKey(tokenHash))
}
