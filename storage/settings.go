package storage

import (
	"database/sql"
	"encoding/json"
	"log"
)

func defaultSettings() map[string]any {
	return map[string]any{
		"filter":            "",
		"feed":              "",
		"feed_list_width":   300,
		"item_list_width":   300,
		"sort_newest_first": true,
		"theme_name":        "light",
		"theme_font":        "",
		"theme_size":        1,
		"refresh_rate":      0,
	}
}

func (s *Storage) GetSettingsValue(key string) (any, error) {
	row := s.db.QueryRow(`select val from settings where key = ?`, key)
	var val []byte
	err := row.Scan(&val)
	if err != nil {
		if err == sql.ErrNoRows {
			return defaultSettings()[key], nil
		}
		return nil, err
	}
	if len(val) == 0 {
		return nil, nil
	}
	var valDecoded any
	if err := json.Unmarshal(val, &valDecoded); err != nil {
		return nil, err
	}
	return valDecoded, nil
}

func (s *Storage) GetSettingsValueInt64(key string) (int64, error) {
	val, err := s.GetSettingsValue(key)
	if err != nil {
		log.Print(err)
		return 0, err
	} else if fval, ok := val.(float64); ok {
		return int64(fval), nil
	}
	return 0, nil
}

func (s *Storage) GetSettings() (map[string]any, error) {
	result := defaultSettings()
	rows, err := s.db.Query(`select key, val from settings`)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	for rows.Next() {
		var key string
		var val []byte
		var valDecoded any

		if err = rows.Scan(&key, &val); err != nil {
			log.Print(err)
			return nil, err
		} else if err = json.Unmarshal(val, &valDecoded); err != nil {
			log.Print(err)
			return nil, err
		}
		result[key] = valDecoded
	}
	return result, nil
}

func (s *Storage) UpdateSettings(kv map[string]any) error {
	defaults := defaultSettings()
	for key, val := range kv {
		if _, ok := defaults[key]; !ok {
			continue
		}
		valEncoded, err := json.Marshal(val)
		if err != nil {
			log.Print(err)
			return err
		}
		_, err = s.db.Exec(`
			insert into settings (key, val) values (?, ?)
			on conflict (key) do update set val = ?`,
			key, valEncoded, valEncoded,
		)
		if err != nil {
			log.Print(err)
			return err
		}
	}
	return nil
}
