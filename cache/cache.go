package cache

import (
	"time"

	"golang.org/x/sync/singleflight"
)

type ICache interface {
	Get(key string, ex bool, ttl time.Duration) (any, bool, error)
	Set(key string, value any, ttl time.Duration) error
}

type Cache struct {
	ICache
	g *singleflight.Group
}

func NewCache(c ICache) *Cache {
	return &Cache{c, new(singleflight.Group)}
}

func (c *Cache) TryGet(key string, ttl time.Duration, ex bool, f func() (any, error)) (any, error) {
	val, ok, err := c.Get(key, ex, ttl)
	if err != nil {
		return nil, err
	} else if ok {
		return val, nil
	}
	val, err, _ = c.g.Do(key, f)
	if err != nil {
		return nil, err
	}
	return val, c.Set(key, val, ttl)
}
