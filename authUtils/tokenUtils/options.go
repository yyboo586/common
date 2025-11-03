package tokenUtils

import (
	"time"

	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/yyboo586/common/cacheUtils"
)

var (
	defaultToken = Token{
		ServerName: "defaultGFToken",
		CacheKey:   "defaultGFToken_",
		Timeout:    60 * 60 * 24 * 10,
		MaxRefresh: 60 * 60 * 24 * 5,
		cache:      cacheUtils.NewMemory("defaultGFToken_"),
		userJwt:    CreateMyJWT("defaultGFToken"),
		MultiLogin: false,
		EncryptKey: []byte("49c54195e750b04e74a8429b17aefc77"),
	}
)

type OptionFunc func(*Token)

func NewToken(opts ...OptionFunc) *Token {
	g := defaultToken
	for _, o := range opts {
		o(&g)
	}
	return &g
}

func WithAccessTokenLifeSpan(value time.Duration) OptionFunc {
	return func(g *Token) {
		g.AccessTokenLifeSpan = value
	}
}

func WithRefreshTokenLifeSpan(value time.Duration) OptionFunc {
	return func(g *Token) {
		g.RefreshTokenLifeSpan = value
	}
}

func WithCacheKey(value string) OptionFunc {
	return func(g *Token) {
		g.CacheKey = value
	}
}

func WithRedisCache() OptionFunc {
	return func(t *Token) {
		t.cache = cacheUtils.NewRedis(t.CacheKey)
	}
}

func WithExcludePaths(value g.SliceStr) OptionFunc {
	return func(g *Token) {
		g.ExcludePaths = value
	}
}
