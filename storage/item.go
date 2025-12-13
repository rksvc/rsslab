package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"rsslab/utils"
	"slices"
	"strings"
	"time"
)

type ItemStatus int

const (
	UNREAD ItemStatus = iota
	READ
	STARRED
)

var StatusRepresentations = map[ItemStatus]string{
	UNREAD:  "unread",
	READ:    "read",
	STARRED: "starred",
}

var StatusValues = map[string]ItemStatus{
	"unread":  UNREAD,
	"read":    READ,
	"starred": STARRED,
}

var errInvalidValue = fmt.Errorf("invalid value for %T", ItemStatus(0))

func (s ItemStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(StatusRepresentations[s])
}

func (s *ItemStatus) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	v, ok := StatusValues[str]
	if !ok {
		return errInvalidValue
	}
	*s = v
	return nil
}

func (s *ItemStatus) UnmarshalText(text []byte) error {
	v, ok := StatusValues[utils.BytesToString(text)]
	if !ok {
		return errInvalidValue
	}
	*s = v
	return nil
}

type Item struct {
	Id       int        `json:"id"`
	GUID     string     `json:"guid"`
	FeedId   int        `json:"feed_id"`
	Title    string     `json:"title"`
	Link     string     `json:"link"`
	Content  string     `json:"content"`
	Date     time.Time  `json:"date"`
	Status   ItemStatus `json:"status"`
	ImageURL *string    `json:"image,omitempty"`
	AudioURL *string    `json:"podcast_url,omitempty"`
}

func (s *Storage) CreateItems(items []Item, feedId int, lastRefreshed time.Time, state *HTTPState) error {
	tx, err := s.db.Begin()
	if err != nil {
		return newError(err)
	}

	slices.SortStableFunc(items, func(a, b Item) int {
		return b.Date.Compare(a.Date)
	})
	lastRefreshed = lastRefreshed.UTC()
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		_, err := tx.Exec(`
			insert into items (
				guid, feed_id, title, link, date,
				content, content_text, image,
				podcast_url, date_arrived, status
			)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			on conflict (feed_id, guid) do nothing`,
			item.GUID, item.FeedId, item.Title, item.Link, item.Date.UTC(),
			item.Content, utils.ExtractText(item.Content), item.ImageURL,
			item.AudioURL, lastRefreshed, UNREAD,
		)
		if err != nil {
			if err := tx.Rollback(); err != nil {
				log.Print(err)
			}
			return newError(err)
		}
	}

	acts := []string{"last_refreshed = ?"}
	args := []any{lastRefreshed}
	if len(items) > 0 {
		acts = append(acts, "size = max(ifnull(size, 0), ?)")
		args = append(args, len(items))
	}
	if state != nil {
		acts = append(acts, "last_modified = ?")
		args = append(args, state.LastModified)
		acts = append(acts, "etag = ?")
		args = append(args, state.Etag)
	}
	args = append(args, feedId)
	_, err = tx.Exec(fmt.Sprintf(`
		update feeds set %s
		where id = ?
	`, strings.Join(acts, ", ")), args...)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			log.Print(err)
		}
		return newError(err)
	}

	err = tx.Commit()
	if err != nil {
		return newError(err)
	}
	return nil
}

type ItemFilter struct {
	FolderId    *int        `json:"folder_id"`
	FeedId      *int        `json:"feed_id"`
	Status      *ItemStatus `json:"status"`
	Search      *string     `json:"search"`
	After       *int        `json:"after"`
	OldestFirst bool        `json:"oldest_first"`
}

func listQueryPredicate(filter ItemFilter, includeBoundary bool) (string, []any) {
	var cond []string
	var args []any
	if filter.FolderId != nil {
		cond = append(cond, "feed_id in (select id from feeds where folder_id = ?)")
		args = append(args, *filter.FolderId)
	}
	if filter.FeedId != nil {
		cond = append(cond, "feed_id = ?")
		args = append(args, *filter.FeedId)
	}
	if filter.Status != nil {
		cond = append(cond, "status = ?")
		args = append(args, *filter.Status)
	}
	if filter.Search != nil {
		for _, word := range strings.Fields(*filter.Search) {
			word = "%" + word + "%"
			cond = append(cond, "(title like ? or content_text like ?)")
			args = append(args, word, word)
		}
	}
	if filter.After != nil {
		compare := "<"
		if filter.OldestFirst {
			compare = ">"
		}
		if includeBoundary {
			compare += "="
		}
		cond = append(cond, fmt.Sprintf("(date, id) %s (select date, id from items where id = ?)", compare))
		args = append(args, *filter.After)
	}

	predicate := "1"
	if len(cond) > 0 {
		predicate = strings.Join(cond, " and ")
	}
	return predicate, args
}

func (s *Storage) ListItems(filter ItemFilter, limit int) ([]Item, error) {
	predicate, args := listQueryPredicate(filter, false)
	order := "date desc, id desc"
	if filter.OldestFirst {
		order = "date asc, id asc"
	}
	rows, err := s.db.Query(fmt.Sprintf(`
		select
			id, guid, feed_id, iif(title = '', content, title),
			link, date, status, image, podcast_url
		from items
		where %s
		order by %s
		limit %d
	`, predicate, order, limit), args...)
	if err != nil {
		return nil, newError(err)
	}
	result := make([]Item, 0)
	for rows.Next() {
		var i Item
		err = rows.Scan(
			&i.Id, &i.GUID, &i.FeedId,
			&i.Title, &i.Link, &i.Date,
			&i.Status, &i.ImageURL, &i.AudioURL,
		)
		if err != nil {
			return nil, newError(err)
		}
		result = append(result, i)
	}
	return result, nil
}

func (s *Storage) GetItem(id int) (*Item, error) {
	var i Item
	err := s.db.QueryRow(`
		select
			id, guid, feed_id, title, link, content,
			date, status, image, podcast_url
		from items
		where id = ?
	`, id).Scan(
		&i.Id, &i.GUID, &i.FeedId, &i.Title, &i.Link, &i.Content,
		&i.Date, &i.Status, &i.ImageURL, &i.AudioURL,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, newError(err)
	}
	return &i, nil
}

func (s *Storage) UpdateItemStatus(itemId int, status ItemStatus) error {
	_, err := s.db.Exec(`update items set status = ? where id = ?`, status, itemId)
	if err != nil {
		return newError(err)
	}
	return nil
}

func (s *Storage) MarkItemsRead(filter ItemFilter) error {
	predicate, args := listQueryPredicate(filter, true)
	_, err := s.db.Exec(fmt.Sprintf(`
		update items set status = %d
		where %s and status != %d
	`, READ, predicate, STARRED), args...)
	if err != nil {
		return newError(err)
	}
	return nil
}

const (
	ITEMS_KEEP_SIZE = 50
	ITEMS_KEEP_DAYS = 90
)

// Delete old articles from the database to clean up space.
//
// The rules:
//   - Never delete unread/starred entries.
//   - Keep at least the same amount of articles the feed provides (default: 50).
//     This prevents from deleting items for rarely updated and/or ever-growing
//     feeds which might eventually reappear as unread.
//   - Keep entries for a certain period (default: 90 days).
func (s *Storage) DeleteOldItems() {
	rows, err := s.db.Query(`
		select id, max(ifnull(size, 0), ?)
		from feeds
	`, ITEMS_KEEP_SIZE)
	if err != nil {
		log.Print(err)
		return
	}
	feedLimits := make(map[int]int)
	for rows.Next() {
		var feedId, limit int
		err = rows.Scan(&feedId, &limit)
		if err != nil {
			log.Print(err)
			return
		}
		feedLimits[feedId] = limit
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return
	}

	dateArrived := time.Now().Add(-time.Duration(ITEMS_KEEP_DAYS) * 24 * time.Hour).UTC()
	for feedId, limit := range feedLimits {
		result, err := s.db.Exec(`
			delete from items
			where id in (
				select id
				from items
				where feed_id = ? and status = ?
				order by date desc
				limit -1 offset ?
			) and date_arrived < ?
			`,
			feedId,
			READ,
			limit,
			dateArrived,
		)
		if err != nil {
			log.Print(err)
			return
		}
		numDeleted, err := result.RowsAffected()
		if err != nil {
			log.Print(err)
		} else if numDeleted > 0 {
			log.Printf("deleted %d old items (feed: %d)", numDeleted, feedId)
		}
	}
}
