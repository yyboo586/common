package cacheUtils

import (
	"context"
	"time"

	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/os/gcache"
)

type ICache interface {
	Get(ctx context.Context, key string) *gvar.Var
	Set(ctx context.Context, key string, value interface{}, duration time.Duration, tag ...string)
	Remove(ctx context.Context, key string) *gvar.Var
	Removes(ctx context.Context, keys []string)
	RemoveByTag(ctx context.Context, tag string)
	RemoveByTags(ctx context.Context, tag []string)
	SetIfNotExist(ctx context.Context, key string, value interface{}, duration time.Duration, tag string) bool
	GetOrSet(ctx context.Context, key string, value interface{}, duration time.Duration, tag string) *gvar.Var
	GetOrSetFunc(ctx context.Context, key string, f gcache.Func, duration time.Duration, tag string) *gvar.Var
	GetOrSetFuncLock(ctx context.Context, key string, f gcache.Func, duration time.Duration, tag string) *gvar.Var
	Contains(ctx context.Context, key string) bool
	Data(ctx context.Context) map[interface{}]interface{}
	Keys(ctx context.Context) []interface{}
	KeyStrings(ctx context.Context) []string
	Values(ctx context.Context) []interface{}
	Size(ctx context.Context) int
}
