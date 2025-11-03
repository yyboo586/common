package cacheUtils

import (
	"context"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/gogf/gf/v2/container/gvar"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcache"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/gogf/gf/v2/util/gconv"
)

var logger *glog.Logger

type Cache struct {
	CachePrefix string //缓存前缀
	cache       *gcache.Cache
	tagSetMux   sync.Mutex
}

func init() {
	logger = glog.New()
	logger.SetLevel(glog.LEVEL_ALL)
	logger.SetPrefix("[cacheUtils]")
	logger.SetTimeFormat(time.DateTime)
	logger.SetWriter(os.Stdout)
}

// New 使用内存缓存
func NewMemory(cachePrefix string) ICache {
	cache := &Cache{
		CachePrefix: cachePrefix,
		cache:       gcache.New(),
	}
	return cache
}

// NewRedis 使用redis缓存
func NewRedis(cachePrefix string, redisName ...string) ICache {
	cache := &Cache{
		CachePrefix: cachePrefix,
		cache:       gcache.NewWithAdapter(gcache.NewAdapterRedis(g.Redis(redisName...))),
	}
	return cache
}

// 设置tag缓存的keys
func (c *Cache) cacheTagKey(ctx context.Context, key interface{}, tag string) {
	tagKey := c.CachePrefix + c.setTagKey(tag)
	if tagKey != "" {
		tagValue := []interface{}{key}
		value, _ := c.cache.Get(ctx, tagKey)
		if !value.IsNil() {
			var keyValue []interface{}
			//若是字符串
			if kStr, ok := value.Val().(string); ok {
				js, err := gjson.DecodeToJson(kStr)
				if err != nil {
					logger.Error(ctx, err)
					return
				}
				keyValue = gconv.SliceAny(js.Interface())
			} else {
				keyValue = gconv.SliceAny(value)
			}
			for _, v := range keyValue {
				if !reflect.DeepEqual(key, v) {
					tagValue = append(tagValue, v)
				}
			}
		}
		c.cache.Set(ctx, tagKey, tagValue, 0)
	}
}

// 获取带标签的键名
func (c *Cache) setTagKey(tag string) string {
	if tag != "" {
		tag = "tag_" + tag
	}
	return tag
}

// Set sets cache with <tagKey>-<value> pair, which is expired after <duration>.
// It does not expire if <duration> <= 0.
func (c *Cache) Set(ctx context.Context, key string, value interface{}, duration time.Duration, tag ...string) {
	c.tagSetMux.Lock()
	if len(tag) > 0 {
		c.cacheTagKey(ctx, key, tag[0])
	}
	err := c.cache.Set(ctx, c.CachePrefix+key, value, duration)
	if err != nil {
		logger.Error(ctx, err)
	}
	c.tagSetMux.Unlock()
}

// SetIfNotExist sets cache with <tagKey>-<value> pair if <tagKey> does not exist in the cache,
// which is expired after <duration>. It does not expire if <duration> <= 0.
func (c *Cache) SetIfNotExist(ctx context.Context, key string, value interface{}, duration time.Duration, tag string) bool {
	c.tagSetMux.Lock()
	defer c.tagSetMux.Unlock()

	c.cacheTagKey(ctx, key, tag)
	v, err := c.cache.SetIfNotExist(ctx, c.CachePrefix+key, value, duration)
	if err != nil {
		logger.Error(ctx, err)
	}
	return v
}

// Get returns the value of <tagKey>.
// It returns nil if it does not exist or its value is nil.
func (c *Cache) Get(ctx context.Context, key string) *gvar.Var {
	v, err := c.cache.Get(ctx, c.CachePrefix+key)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Get cache: %v", key)
	return v
}

// GetOrSet returns the value of <tagKey>,
// or sets <tagKey>-<value> pair and returns <value> if <tagKey> does not exist in the cache.
// The tagKey-value pair expires after <duration>.
//
// It does not expire if <duration> <= 0.
func (c *Cache) GetOrSet(ctx context.Context, key string, value interface{}, duration time.Duration, tag string) *gvar.Var {
	c.tagSetMux.Lock()
	defer c.tagSetMux.Unlock()
	c.cacheTagKey(ctx, key, tag)
	v, err := c.cache.GetOrSet(ctx, c.CachePrefix+key, value, duration)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "GetOrSet cache: %v", key)
	return v
}

// GetOrSetFunc returns the value of <tagKey>, or sets <tagKey> with result of function <f>
// and returns its result if <tagKey> does not exist in the cache. The tagKey-value pair expires
// after <duration>. It does not expire if <duration> <= 0.
func (c *Cache) GetOrSetFunc(ctx context.Context, key string, f gcache.Func, duration time.Duration, tag string) *gvar.Var {
	c.tagSetMux.Lock()
	defer c.tagSetMux.Unlock()

	c.cacheTagKey(ctx, key, tag)
	v, err := c.cache.GetOrSetFunc(ctx, c.CachePrefix+key, f, duration)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "GetOrSetFunc cache: %v", key)
	return v
}

// GetOrSetFuncLock returns the value of <tagKey>, or sets <tagKey> with result of function <f>
// and returns its result if <tagKey> does not exist in the cache. The tagKey-value pair expires
// after <duration>. It does not expire if <duration> <= 0.
//
// Note that the function <f> is executed within writing mutex lock.
func (c *Cache) GetOrSetFuncLock(ctx context.Context, key string, f gcache.Func, duration time.Duration, tag string) *gvar.Var {
	c.tagSetMux.Lock()
	defer c.tagSetMux.Unlock()

	c.cacheTagKey(ctx, key, tag)
	v, err := c.cache.GetOrSetFuncLock(ctx, c.CachePrefix+key, f, duration)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "GetOrSetFuncLock cache: %v", key)
	return v
}

// Contains returns true if <tagKey> exists in the cache, or else returns false.
func (c *Cache) Contains(ctx context.Context, key string) bool {
	v, err := c.cache.Contains(ctx, c.CachePrefix+key)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Contains cache: %v", key)
	return v
}

// Remove deletes the <tagKey> in the cache, and returns its value.
func (c *Cache) Remove(ctx context.Context, key string) *gvar.Var {
	v, err := c.cache.Remove(ctx, c.CachePrefix+key)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Remove cache: %v", key)
	return v
}

// Removes deletes <keys> in the cache.
func (c *Cache) Removes(ctx context.Context, keys []string) {
	keysWithPrefix := make([]interface{}, len(keys))
	for k, v := range keys {
		keysWithPrefix[k] = c.CachePrefix + v
	}
	c.cache.Remove(ctx, keysWithPrefix...)
}

// RemoveByTag deletes the <tag> in the cache, and returns its value.
func (c *Cache) RemoveByTag(ctx context.Context, tag string) {
	c.tagSetMux.Lock()
	tagKey := c.setTagKey(tag)
	//删除tagKey 对应的 key和值
	keys := c.Get(ctx, tagKey)
	if !keys.IsNil() {
		//如果是字符串
		if kStr, ok := keys.Val().(string); ok {
			js, err := gjson.DecodeToJson(kStr)
			if err != nil {
				logger.Error(ctx, err)
				return
			}
			ks := gconv.SliceStr(js.Interface())
			c.Removes(ctx, ks)
		} else {
			ks := gconv.SliceStr(keys.Val())
			c.Removes(ctx, ks)
		}
	}
	c.Remove(ctx, tagKey)
	c.tagSetMux.Unlock()
}

// RemoveByTags deletes <tags> in the cache.
func (c *Cache) RemoveByTags(ctx context.Context, tag []string) {
	for _, v := range tag {
		c.RemoveByTag(ctx, v)
	}
}

// Data returns a copy of all tagKey-value pairs in the cache as map type.
func (c *Cache) Data(ctx context.Context) map[interface{}]interface{} {
	v, err := c.cache.Data(ctx)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Data cache: %v", v)
	return v
}

// Keys returns all keys in the cache as slice.
func (c *Cache) Keys(ctx context.Context) []interface{} {
	v, err := c.cache.Keys(ctx)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Keys cache: %v", v)
	return v
}

// KeyStrings returns all keys in the cache as string slice.
func (c *Cache) KeyStrings(ctx context.Context) []string {
	v, err := c.cache.KeyStrings(ctx)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "KeyStrings cache: %v", v)
	return v
}

// Values returns all values in the cache as slice.
func (c *Cache) Values(ctx context.Context) []interface{} {
	v, err := c.cache.Values(ctx)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Values cache: %v", v)
	return v
}

// Size returns the size of the cache.
func (c *Cache) Size(ctx context.Context) int {
	v, err := c.cache.Size(ctx)
	if err != nil {
		logger.Error(ctx, err)
	}
	logger.Debugf(ctx, "Size cache: %v", v)
	return v
}
