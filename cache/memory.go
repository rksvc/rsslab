package cache

import (
	"time"

	"github.com/karlseguin/ccache/v3"
)

type LRU struct {
	c *ccache.Cache[any]
}

func NewLRU() *LRU {
	return &LRU{ccache.New(ccache.Configure[any]().MaxSize(256))}
}

func (m *LRU) Get(key string, _ bool, _ time.Duration) (any, bool, error) {
	item := m.c.Get(key)
	if item == nil || item.Expired() {
		return nil, false, nil
	}
	return item.Value(), true, nil
}

func (m *LRU) Set(key string, value any, ttl time.Duration) error {
	m.c.Set(key, value, ttl)
	return nil
}
