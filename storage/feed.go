package storage

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-errors/errors"
)

type Feed struct {
	Id            int        `json:"id"`
	FolderId      *int       `json:"folder_id"`
	Title         string     `json:"title"`
	Link          string     `json:"link,omitempty"`
	FeedLink      string     `json:"feed_link"`
	HasIcon       bool       `json:"has_icon"`
	LastRefreshed *time.Time `json:"last_refreshed,omitempty"`
}

func (s *Storage) CreateFeed(title, link, feedLink string, folderId *int) (*Feed, error) {
	if title == "" {
		title = feedLink
	}
	var id int
	err := s.db.QueryRow(`
		insert into feeds (title, link, feed_link, folder_id) 
		values (?, ?, ?, ?)
		on conflict (feed_link) do update set folder_id = ?
        returning id`,
		title, link, feedLink, folderId, folderId,
	).Scan(&id)
	if err != nil {
		return nil, errors.New(err)
	}
	return &Feed{
		Id:       id,
		Title:    title,
		Link:     link,
		FeedLink: feedLink,
		FolderId: folderId,
	}, nil
}

func (s *Storage) DeleteFeed(feedId int) error {
	_, err := s.db.Exec(`delete from feeds where id = ?`, feedId)
	if err != nil {
		return errors.New(err)
	}
	return nil
}

type FeedEditor struct {
	Title    *string `json:"title"`
	FeedLink *string `json:"feed_link"`
	FolderId **int   `json:"folder_id"`
}

func (s *Storage) EditFeed(feedId int, editor FeedEditor) error {
	var acts []string
	var args []any
	if editor.Title != nil {
		acts = append(acts, "title = ?")
		args = append(args, *editor.Title)
	}
	if editor.FeedLink != nil {
		acts = append(acts, "feed_link = ?")
		args = append(args, *editor.FeedLink)
	}
	if editor.FolderId != nil {
		acts = append(acts, "folder_id = ?")
		args = append(args, *editor.FolderId)
	}
	if len(acts) == 0 {
		return nil
	}
	args = append(args, feedId)
	_, err := s.db.Exec(fmt.Sprintf(`update feeds set %s where id = ?`, strings.Join(acts, ", ")), args...)
	if err != nil {
		return errors.New(err)
	}
	return nil
}

func (s *Storage) UpdateFeedIcon(feedId int, icon []byte) {
	_, err := s.db.Exec(`update feeds set icon = ? where id = ?`, icon, feedId)
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) ListFeeds() ([]Feed, error) {
	rows, err := s.db.Query(`
		select id, folder_id, title, link, feed_link,
		       last_refreshed, icon is not null as has_icon
		from feeds
		order by title collate nocase
	`)
	if err != nil {
		return nil, errors.New(err)
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.FolderId,
			&f.Title,
			&f.Link,
			&f.FeedLink,
			&f.LastRefreshed,
			&f.HasIcon,
		)
		if err != nil {
			return nil, errors.New(err)
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.New(err)
	}
	return result, nil
}

func (s *Storage) ListFeedsMissingIcons() (result []Feed) {
	rows, err := s.db.Query(`
		select id, link, feed_link
		from feeds
		where icon is null
	`)
	if err != nil {
		log.Print(err)
		return
	}
	for rows.Next() {
		var f Feed
		err = rows.Scan(
			&f.Id,
			&f.Link,
			&f.FeedLink,
		)
		if err != nil {
			log.Print(err)
			return
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return
	}
	return
}

func (s *Storage) GetFeed(id int) (*Feed, error) {
	var f Feed
	err := s.db.QueryRow(`
		select id, link, feed_link
		from feeds where id = ?
	`, id).Scan(&f.Id, &f.Link, &f.FeedLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.New(err)
	}
	return &f, nil
}

func (s *Storage) GetFeedIcon(id int) ([]byte, error) {
	var icon []byte
	err := s.db.QueryRow(`
		select icon from feeds where id = ?
	`, id).Scan(&icon)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.New(err)
	}
	return icon, nil
}

func (s *Storage) GetFeeds(folderId int) ([]Feed, error) {
	rows, err := s.db.Query(`
		select id, feed_link
		from feeds
		where folder_id = ?
		order by title collate nocase
	`, folderId)
	if err != nil {
		return nil, errors.New(err)
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(&f.Id, &f.FeedLink)
		if err != nil {
			return nil, errors.New(err)
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		return nil, errors.New(err)
	}
	return result, nil
}

func (s *Storage) SetFeedError(feedId int, lastError error) {
	var val any
	if lastError != nil {
		val = lastError.Error()
	}
	_, err := s.db.Exec(`
		update feeds set error = ? where id = ?`,
		val, feedId,
	)
	if err != nil {
		log.Print(err)
	}
}

func (s *Storage) GetFeedErrors() (map[int]string, error) {
	rows, err := s.db.Query(`
		select id, error from feeds where error is not null`,
	)
	if err != nil {
		return nil, errors.New(err)
	}

	errs := make(map[int]string)
	for rows.Next() {
		var id int
		var err string
		if err := rows.Scan(&id, &err); err != nil {
			return nil, errors.New(err)
		}
		errs[id] = err
	}
	if err = rows.Err(); err != nil {
		return nil, errors.New(err)
	}
	return errs, nil
}

type HTTPState struct {
	LastModified *string
	Etag         *string
}

func (s *Storage) GetHTTPState(feedId int) (state HTTPState, err error) {
	err = s.db.QueryRow(`
		select last_modified, etag
		from feeds where id = ?
	`, feedId).Scan(
		&state.LastModified,
		&state.Etag,
	)
	if err != nil {
		err = errors.New(err)
	}
	return
}
