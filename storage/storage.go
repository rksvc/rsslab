package storage

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/sync/singleflight"
)

type Storage struct {
	db    *sql.DB
	group singleflight.Group
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	if err = migrate(db); err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Optimize() {
	_, err := s.db.Exec("pragma optimize")
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) Vacuum() {
	_, err := s.db.Exec("pragma incremental_vacuum")
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) PurgeCache() {
	now := time.Now().UTC()
	for _, table := range []string{"src_cache", "content_cache"} {
		_, err := s.db.Exec(fmt.Sprintf(`delete from %s where expire < ?`, table), now)
		if err != nil {
			log.Print(err)
		}
	}
}
