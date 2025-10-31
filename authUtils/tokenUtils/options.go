package tokenUtils

import (
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/database/gredis"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/yyboo586/common/authUtils/adapter"
)

var (
	defaultToken = Token{
		ServerName: "defaultGFToken",
		CacheKey:   "defaultGFToken_",
		Timeout:    60 * 60 * 24 * 10,
		MaxRefresh: 60 * 60 * 24 * 5,
		cache:      gcache.New(),
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

func WithExcludePaths(value g.SliceStr) OptionFunc {
	return func(g *Token) {
		g.ExcludePaths = value
	}
}

func WithEncryptKey(value []byte) OptionFunc {
	return func(g *Token) {
		g.EncryptKey = value
	}
}

func WithServerName(value string) OptionFunc {
	return func(g *Token) {
		g.ServerName = value
	}
}

func WithCacheKey(value string) OptionFunc {
	return func(g *Token) {
		g.CacheKey = value
	}
}

func WithTimeoutAndMaxRefresh(timeout, maxRefresh int64) OptionFunc {
	return func(g *Token) {
		g.Timeout = timeout
		g.MaxRefresh = maxRefresh
	}
}

func WithTimeout(value int64) OptionFunc {
	return func(g *Token) {
		g.Timeout = value
	}
}

func WithMaxRefresh(value int64) OptionFunc {
	return func(g *Token) {
		g.MaxRefresh = value
	}
}

func WithUserJwt(key string) OptionFunc {
	return func(g *Token) {
		g.userJwt = CreateMyJWT(key)
	}
}

func WithGCache() OptionFunc {
	return func(g *Token) {
		g.cache = gcache.New()
	}
}

func WithGRedis(redis ...*gredis.Redis) OptionFunc {
	return func(gf *Token) {
		gf.cache = gcache.New()
		if len(redis) > 0 {
			gf.cache.SetAdapter(gcache.NewAdapterRedis(redis[0]))
		} else {
			gf.cache.SetAdapter(gcache.NewAdapterRedis(g.Redis()))
		}
	}
}

func WithDist(dist ...*adapter.Dist) OptionFunc {
	return func(gf *Token) {
		gf.cache = gcache.New()
		if len(dist) > 0 {
			gf.cache.SetAdapter(dist[0])
		} else {
			gf.cache.SetAdapter(adapter.NewDist())
		}
	}
}

func WithGRedisConfig(redisConfig ...*gredis.Config) OptionFunc {
	return func(g *Token) {
		g.cache = gcache.New()
		redis, err := gredis.New(redisConfig...)
		if err != nil {
			panic(err)
		}
		g.cache.SetAdapter(gcache.NewAdapterRedis(redis))
	}
}

func WithDistConfig(distConfig *adapter.Config) OptionFunc {
	return func(g *Token) {
		g.cache = gcache.New()
		adapter.SetConfig(distConfig)
		dist := adapter.New()
		g.cache.SetAdapter(dist)
	}
}

func WithMultiLogin(b bool) OptionFunc {
	return func(g *Token) {
		g.MultiLogin = b
	}
}
