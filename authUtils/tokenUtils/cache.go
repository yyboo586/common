package tokenUtils

import (
	"context"
	"time"
)

func (m *Token) cacheContains(ctx context.Context, key string) bool {
	ok := m.cache.Contains(ctx, key)
	return ok
}

func (m *Token) setCache(ctx context.Context, key string, value interface{}, ttl time.Duration) (err error) {
	m.cache.Set(ctx, key, value, ttl)
	return
}

func (m *Token) getCache(ctx context.Context, key string) (out string, err error) {
	result := m.cache.Get(ctx, key)

	if result.Val() != nil {
		out = result.String()
	}
	return
}

func (m *Token) removeCache(ctx context.Context, key string) (err error) {
	_, err = m.cache.Remove(ctx, key)
	return
}
