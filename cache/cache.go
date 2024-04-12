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
	ttl time.Duration
	g   *singleflight.Group
}

func NewCache(c ICache, ttl time.Duration) *Cache {
	return &Cache{c, ttl, new(singleflight.Group)}
}

func (c *Cache) TryGet(key string, ex bool, f func() (any, error)) (any, error) {
	val, ok, err := c.Get(key, ex, c.ttl)
	if err != nil {
		return nil, err
	} else if ok {
		return val, nil
	}
	val, err, _ = c.g.Do(key, f)
	if err != nil {
		return nil, err
	}
	return val, c.Set(key, val, c.ttl)
}
