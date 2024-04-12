package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Redis struct {
	c *redis.Client
}

func NewRedis(redisUrl string) *Redis {
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Fatal(err)
	}
	return &Redis{redis.NewClient(opt)}
}

func (r *Redis) Get(key string, ex bool, ttl time.Duration) (v any, ok bool, err error) {
	if ex {
		v, err = r.c.GetEx(context.Background(), key, ttl).Bytes()
	} else {
		v, err = r.c.Get(context.Background(), key).Bytes()
	}
	if ok = err == nil; ok {
		return
	} else if err == redis.Nil {
		err = nil
	}
	return
}

func (r *Redis) Set(key string, value any, ttl time.Duration) error {
	switch value.(type) {
	case string, []byte:
	default:
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}
		value = b
	}
	return r.c.SetEx(context.Background(), key, value, ttl).Err()
}
