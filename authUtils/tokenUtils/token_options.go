package tokenUtils

import (
	"context"
	"os"
	"time"

	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"
)

var (
	defaultToken = Token{
		AccessTokenTimeout: 24 * time.Hour,     // 访问令牌过期时间 1天
		RefreshTimeout:     5 * 24 * time.Hour, // 刷新令牌过期时间 5天
		signer:             CreateMyJWT("defaultTokenComponent"),
	}
)

func init() {
	defaultToken.logger = glog.New()
	defaultToken.logger.SetLevel(glog.LEVEL_ALL)
	defaultToken.logger.SetPrefix("[tokenUtils]")
	defaultToken.logger.SetTimeFormat(time.DateTime)
	defaultToken.logger.SetWriter(os.Stdout)
}

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

func WithTokenStoreConfig(cfg *TokenStoreConfig) OptionFunc {
	return func(g *Token) {
		store, err := NewTokenStore(cfg)
		if err != nil {
			panic(err)
		}
		if err := store.EnsureTable(context.Background()); err != nil {
			panic(err)
		}
		g.store = store
	}
}
