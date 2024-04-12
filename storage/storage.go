package storage

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

func New(path string) (*Storage, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
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
