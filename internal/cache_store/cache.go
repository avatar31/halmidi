package cachestore

import (
	"context"
	"sync"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/dustin/go-humanize"

	"github.com/avatar31/halmidi/config"
	"github.com/avatar31/halmidi/internal/logger"
)

type Cache struct {
	cache *ristretto.Cache[string, any]
}

const (
	DefaultCacheSize = 1 << 27 // 128MB
)

var (
	cacheStore *Cache
	once       sync.Once
)

func InitCacheStore(ctx context.Context) {
	once.Do(func() {
		size := config.GetConfig().Cache.Size
		if size <= 0 {
			size = DefaultCacheSize
		}

		var cache *ristretto.Cache[string, any]
		cache, err := ristretto.NewCache(&ristretto.Config[string, any]{
			NumCounters: 1e7, // number of keys to track frequency of (10M).
			MaxCost:     size,
			BufferItems: 64, // number of keys per Get buffer.
			Metrics:     true,
		})
		if err != nil {
			logger.GetLogger(ctx).WithError(err).Panic("Failed to initialize cache store")
		}

		cacheStore = &Cache{cache: cache}
		logger.GetLogger(ctx).Infof("Cache store initialized with size: %s", humanize.IBytes(uint64(size)))
	})
}

func GetCacheStore() *Cache {
	return cacheStore
}

// TODO: P1: We can implement a better cost estimation based on the type and size of value.
func (cs *Cache) Set(key string, value any, ttl time.Duration, cost ...int64) bool {
	c := int64(1)
	if len(cost) > 0 {
		c = cost[0]
	}

	ok := cs.cache.SetWithTTL(key, value, c, ttl)
	if !ok {
		return false
	}
	cs.cache.Wait()
	return true
}

func (cs *Cache) Get(key string) (any, bool) {
	return cs.cache.Get(key)
}

func (cs *Cache) Delete(key string) {
	cs.cache.Del(key)
}

func (cs *Cache) Metrics() string {
	return cs.cache.Metrics.String()
}

func (cs *Cache) Close() {
	cs.cache.Close()
}
