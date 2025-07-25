package storage

import (
	"database/sql"
	"fmt"
	"log"
	"rsslab/utils"
	"strings"
	"time"
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
		return nil, utils.NewError(err)
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
		return utils.NewError(err)
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
		return utils.NewError(err)
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
		select
			id, folder_id, title, link, feed_link,
			icon is not null as has_icon
		from feeds
		order by title collate nocase
	`)
	if err != nil {
		return nil, utils.NewError(err)
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
			&f.HasIcon,
		)
		if err != nil {
			return nil, utils.NewError(err)
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		return nil, utils.NewError(err)
	}
	return result, nil
}

func (s *Storage) ListFeedsMissingIcons() (feeds []Feed) {
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
		feeds = append(feeds, f)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
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
		return nil, utils.NewError(err)
	}
	return &f, nil
}

func (s *Storage) GetFeedIcon(id int) ([]byte, error) {
	var icon []byte
	err := s.db.QueryRow(`select icon from feeds where id = ?`, id).Scan(&icon)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, utils.NewError(err)
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
		return nil, utils.NewError(err)
	}
	result := make([]Feed, 0)
	for rows.Next() {
		var f Feed
		err = rows.Scan(&f.Id, &f.FeedLink)
		if err != nil {
			return nil, utils.NewError(err)
		}
		result = append(result, f)
	}
	if err = rows.Err(); err != nil {
		return nil, utils.NewError(err)
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
		err = utils.NewError(err)
	}
	return
}

type FeedState struct {
	Unread        int        `json:"unread"`
	Starred       int        `json:"starred"`
	LastRefreshed *time.Time `json:"last_refreshed,omitempty"`
	Error         *string    `json:"error,omitempty"`
}

func (s *Storage) FeedState() (map[int]FeedState, error) {
	rows, err := s.db.Query(fmt.Sprintf(`
		select
			feed_id,
			sum(iif(status = %d, 1, 0)),
			sum(iif(status = %d, 1, 0))
		from items
		group by feed_id
	`, UNREAD, STARRED))
	if err != nil {
		return nil, utils.NewError(err)
	}

	result := make(map[int]FeedState)
	for rows.Next() {
		var id int
		var s FeedState
		err = rows.Scan(&id, &s.Unread, &s.Starred)
		if err != nil {
			return nil, utils.NewError(err)
		}
		result[id] = s
	}
	if err = rows.Err(); err != nil {
		return nil, utils.NewError(err)
	}

	rows, err = s.db.Query(`select id, last_refreshed, error from feeds`)
	if err != nil {
		return nil, utils.NewError(err)
	}

	for rows.Next() {
		var id int
		var lastRefreshed *time.Time
		var error *string
		if err = rows.Scan(&id, &lastRefreshed, &error); err != nil {
			return nil, utils.NewError(err)
		}
		if state, ok := result[id]; ok {
			state.LastRefreshed = lastRefreshed
			state.Error = error
			result[id] = state
		}
	}
	if err = rows.Err(); err != nil {
		return nil, utils.NewError(err)
	}

	return result, nil
}
