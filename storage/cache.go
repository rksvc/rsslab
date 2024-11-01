package storage

import (
	"database/sql"
	"fmt"
	"rsslab/utils"
	"time"
)

type CacheType int

const (
	SRC CacheType = iota
	CONTENT
)

func (s *Storage) TryGet(ty CacheType, key string, ttl time.Duration, ex bool, f func() ([]byte, error)) (string, error) {
	var table string
	switch ty {
	case SRC:
		table = "src_cache"
	case CONTENT:
		table = "content_cache"
	}

	expire := time.Now().UTC().Add(ttl)
	var val string
	err := s.db.QueryRow(fmt.Sprintf(`
		update %s
		set expire = iif(?, ?, expire)
		where key = ?
		returning val`, table),
		ex, expire, key,
	).Scan(&val)
	if err == nil {
		return val, nil
	} else if err != sql.ErrNoRows {
		return "", utils.NewError(err)
	}

	v, err, _ := s.group.Do(key, func() (any, error) {
		v, err := f()
		if err != nil {
			return nil, err
		}
		val := utils.BytesToString(v)
		_, err = s.db.Exec(fmt.Sprintf(`
			insert or replace into %s (key, val, expire)
			values (?, ?, ?)`, table),
			key, val, expire,
		)
		return val, err
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}
