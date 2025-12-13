package storage

import (
	"database/sql"
)

const (
	REFRESH_RATE = "refresh_rate"
	DARK_THEME   = "dark_theme"
)

func (s *Storage) GetSettings() (map[string]any, error) {
	rows, err := s.db.Query(`select key, val from settings`)
	if err != nil {
		return nil, newError(err)
	}
	result := make(map[string]any)
	for rows.Next() {
		var key string
		var val any
		err = rows.Scan(
			&key,
			&val,
		)
		if err != nil {
			return nil, newError(err)
		}
		result[key] = val
	}
	if err = rows.Err(); err != nil {
		return nil, newError(err)
	}
	return result, nil
}

func (s *Storage) GetSettingInt(key string) (val *int, err error) {
	err = s.db.QueryRow(`select val from settings where key = ?`, key).Scan(&val)
	if err == sql.ErrNoRows {
		err = nil
	} else if err != nil {
		err = newError(err)
	}
	return
}

func (s *Storage) UpdateSetting(key string, val any) error {
	_, err := s.db.Exec(`
		insert or replace into settings (key, val) values (?, ?)
	`, key, val)
	if err != nil {
		return newError(err)
	}
	return nil
}
