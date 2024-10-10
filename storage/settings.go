package storage

import (
	"database/sql"
	"rsslab/utils"
)

type Settings struct {
	RefreshRate int `json:"refresh_rate"`
}

type SettingsEditor struct {
	RefreshRate *int `json:"refresh_rate"`
}

func (s *Storage) GetSettings() (settings Settings, err error) {
	err = s.db.QueryRow(`
		select val from settings where key = 'refresh_rate'
	`).Scan(&settings.RefreshRate)
	if err != nil {
		if err == sql.ErrNoRows {
			err = nil
		} else {
			err = utils.NewError(err)
		}
	}
	return
}

func (s *Storage) UpdateSettings(settings SettingsEditor) error {
	if settings.RefreshRate != nil {
		_, err := s.db.Exec(`
			insert or replace into settings (key, val) values ('refresh_rate', ?)
		`, *settings.RefreshRate)
		if err != nil {
			return utils.NewError(err)
		}
	}
	return nil
}
