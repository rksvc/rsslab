package storage

import (
	"database/sql"
	"log"
)

type Feed struct {
	Id          int64   `json:"id"`
	FolderId    *int64  `json:"folder_id,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description,omitempty"`
	Link        string  `json:"link,omitempty"`
	FeedLink    string  `json:"feed_link"`
	Icon        *[]byte `json:"icon,omitempty"`
	HasIcon     bool    `json:"has_icon"`
}

type HTTPState struct {
	LastModified *string
	Etag         *string
}

func (s *Storage) CreateFeed(title, description, link, feedLink string, folderId *int64) (*Feed, error) {
	if title == "" {
		title = feedLink
	}
	var id int64
	err := s.db.QueryRow(`
		insert into feeds (title, description, link, feed_link, folder_id) 
		values (?, ?, ?, ?, ?)
		on conflict (feed_link) do update set folder_id = ?
        returning id`,
		title, description, link, feedLink, folderId,
		folderId,
	).Scan(&id)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &Feed{
		Id:          id,
		Title:       title,
		Description: description,
		Link:        link,
		FeedLink:    feedLink,
		FolderId:    folderId,
	}, nil
}

func (s *Storage) DeleteFeed(feedId int64) error {
	_, err := s.db.Exec(`delete from feeds where id = ?`, feedId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) RenameFeed(feedId int64, newTitle string) error {
	_, err := s.db.Exec(`update feeds set title = ? where id = ?`, newTitle, feedId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) EditFeedLink(feedId int64, newFeedLink string) error {
	_, err := s.db.Exec(`update feeds set feed_link = ? where id = ?`, newFeedLink, feedId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) UpdateFeedFolder(feedId int64, newFolderId *int64) error {
	_, err := s.db.Exec(`update feeds set folder_id = ? where id = ?`, newFolderId, feedId)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) UpdateFeedIcon(feedId int64, icon *[]byte) {
	_, err := s.db.Exec(`update feeds set icon = ? where id = ?`, icon, feedId)
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) ListFeeds() ([]Feed, error) {
	rows, err := s.db.Query(`
		select id, folder_id, title, description, link, feed_link,
		       ifnull(length(icon), 0) > 0 as has_icon
		from feeds
		order by title collate nocase
	`)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.FolderId,
			&f.Title,
			&f.Description,
			&f.Link,
			&f.FeedLink,
			&f.HasIcon,
		)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return result, nil
}

func (s *Storage) ListFeedsMissingIcons() []Feed {
	rows, err := s.db.Query(`
		select id, folder_id, title, description, link, feed_link
		from feeds
		where icon is null
	`)
	if err != nil {
		log.Print(err)
		return nil
	}
	var result []Feed
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.FolderId,
			&f.Title,
			&f.Description,
			&f.Link,
			&f.FeedLink,
		)
		if err != nil {
			log.Print(err)
			return nil
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil
	}
	return result
}

func (s *Storage) GetFeed(id int64) (*Feed, error) {
	var f Feed
	err := s.db.QueryRow(`
		select
			id, folder_id, title, link, feed_link,
			icon, ifnull(icon, '') != '' as has_icon
		from feeds where id = ?
	`, id).Scan(
		&f.Id, &f.FolderId, &f.Title, &f.Link, &f.FeedLink,
		&f.Icon, &f.HasIcon,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Print(err)
		return nil, err
	}
	return &f, nil
}

func (s *Storage) GetFeeds(folderId int64) ([]Feed, error) {
	rows, err := s.db.Query(`
		select id, folder_id, title, description, link, feed_link,
		   	ifnull(length(icon), 0) > 0 as has_icon
		from feeds
		where folder_id = ?
		order by title collate nocase
	`, folderId)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.FolderId,
			&f.Title,
			&f.Description,
			&f.Link,
			&f.FeedLink,
			&f.HasIcon,
		)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return result, nil
}

func (s *Storage) ResetFeedError(feedId int64) {
	_, err := s.db.Exec(`
		update feeds set error = null where id = ?`, feedId,
	)
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) SetFeedError(feedId int64, lastError error) {
	_, err := s.db.Exec(`
		update feeds set error = ? where id = ?`,
		lastError.Error(), feedId,
	)
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) GetFeedErrors() (map[int64]string, error) {
	rows, err := s.db.Query(`
		select id, error from feeds where error is not null`,
	)
	if err != nil {
		log.Print(err)
		return nil, err
	}

	errors := make(map[int64]string)
	for rows.Next() {
		var id int64
		var error string
		if err = rows.Scan(&id, &error); err != nil {
			log.Print(err)
			return nil, err
		}
		errors[id] = error
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return errors, nil
}

func (s *Storage) GetFeedsWithErrors() ([]Feed, error) {
	rows, err := s.db.Query(`
		select id, folder_id, title, description, link, feed_link,
		   	ifnull(length(icon), 0) > 0 as has_icon
		from feeds where error is not null
		order by title collate nocase
	`)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.FolderId,
			&f.Title,
			&f.Description,
			&f.Link,
			&f.FeedLink,
			&f.HasIcon,
		)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return result, nil
}

func (s *Storage) SetFeedSize(feedId int64, size int) error {
	_, err := s.db.Exec(`
		update feeds set size = ? where id = ?`,
		size, feedId,
	)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) GetHTTPState(feedId int64) (*HTTPState, error) {
	var state HTTPState
	err := s.db.QueryRow(`
		select last_modified, etag
		from feeds where id = ?
	`, feedId).Scan(
		&state.LastModified,
		&state.Etag,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Print(err)
		return nil, err
	}
	return &state, nil
}

func (s *Storage) SetHTTPState(feedId int64, lastModified, etag string) error {
	_, err := s.db.Exec(`
		update feeds set last_modified = ?, etag = ?
		where id = ?`,
		lastModified, etag, feedId,
	)
	if err != nil {
		log.Print(err)
	}
	return err
}
