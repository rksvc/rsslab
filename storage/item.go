package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/nkanaev/yarr/src/content/htmlutil"
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

func (s ItemStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(StatusRepresentations[s])
}

func (s *ItemStatus) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}
	*s = StatusValues[str]
	return nil
}

type Item struct {
	Id       int64      `json:"id"`
	GUID     string     `json:"guid"`
	FeedId   int64      `json:"feed_id"`
	Title    string     `json:"title"`
	Link     string     `json:"link"`
	Content  string     `json:"content"`
	Date     time.Time  `json:"date"`
	Status   ItemStatus `json:"status"`
	ImageURL *string    `json:"image,omitempty"`
	AudioURL *string    `json:"podcast_url,omitempty"`
}

type ItemFilter struct {
	FolderId *int64
	FeedId   *int64
	Status   *ItemStatus
	Search   *string
	After    *int64
}

type MarkFilter struct {
	FolderId *int64
	FeedId   *int64
}

func (s *Storage) CreateItems(items []Item) error {
	slices.SortStableFunc(items, func(a, b Item) int {
		return b.Date.Compare(a.Date)
	})

	tx, err := s.db.Begin()
	if err != nil {
		log.Print(err)
		return err
	}
	onError := func(err error) error {
		log.Output(2, err.Error())
		if err := tx.Rollback(); err != nil {
			log.Print(err)
		}
		return err
	}
	now := time.Now().UTC()
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		result, err := tx.Exec(`
			insert into items (
				guid, feed_id, title, link, date,
				content, image, podcast_url,
				date_arrived, status
			)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			on conflict (feed_id, guid) do nothing`,
			item.GUID, item.FeedId, item.Title, item.Link, item.Date.UTC(),
			item.Content, item.ImageURL, item.AudioURL,
			now, UNREAD,
		)
		if err != nil {
			return onError(err)
		}
		n, err := result.RowsAffected()
		if err != nil {
			return onError(err)
		}
		if n > 0 {
			itemId, err := result.LastInsertId()
			if err != nil {
				return onError(err)
			}
			result, err := tx.Exec(`
				insert into search (title, description, content) values (?, "", ?)`,
				item.Title, htmlutil.ExtractText(item.Content),
			)
			if err != nil {
				return onError(err)
			}
			rowId, err := result.LastInsertId()
			if err != nil {
				return onError(err)
			}
			_, err = tx.Exec(
				`update items set search_rowid = ? where id = ?`,
				rowId, itemId,
			)
			if err != nil {
				return onError(err)
			}
		}
	}
	if len(items) > 0 {
		_, err = tx.Exec(`
			update feeds set size = ? where id = ?`,
			len(items), items[0].FeedId,
		)
		if err != nil {
			return onError(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Print(err)
	}
	return err
}

func listQueryPredicate(filter ItemFilter, newestFirst bool) (string, []any) {
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
		words := strings.Fields(*filter.Search)
		terms := make([]string, len(words))
		for i, word := range words {
			terms[i] = word + "*"
		}

		cond = append(cond, "search_rowid in (select rowid from search where search match ?)")
		args = append(args, strings.Join(terms, " "))
	}
	if filter.After != nil {
		compare := ">"
		if newestFirst {
			compare = "<"
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

func (s *Storage) ListItems(filter ItemFilter, limit int, newestFirst bool) ([]Item, error) {
	predicate, args := listQueryPredicate(filter, newestFirst)

	order := "date asc, id asc"
	if newestFirst {
		order = "date desc, id desc"
	}

	selectCols := "id, guid, feed_id, title, link, date, status, image, podcast_url"
	query := fmt.Sprintf(`
		select %s
		from items
		where %s
		order by %s
		limit %d
		`, selectCols, predicate, order, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Print(err)
		return nil, err
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
			log.Print(err)
			return nil, err
		}
		result = append(result, i)
	}
	return result, nil
}

func (s *Storage) GetItem(id int64) (*Item, error) {
	i := new(Item)
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
		log.Print(err)
	}
	return i, err
}

func (s *Storage) UpdateItemStatus(item_id int64, status ItemStatus) error {
	_, err := s.db.Exec(`update items set status = ? where id = ?`, status, item_id)
	if err != nil {
		log.Print(err)
	}
	return err
}

func (s *Storage) MarkItemsRead(filter MarkFilter) error {
	predicate, args := listQueryPredicate(ItemFilter{
		FolderId: filter.FolderId,
		FeedId:   filter.FeedId,
	}, false)
	query := fmt.Sprintf(`
		update items set status = %d
		where %s and status != %d
		`, READ, predicate, STARRED)
	_, err := s.db.Exec(query, args...)
	if err != nil {
		log.Print(err)
	}
	return err
}

type FeedStat struct {
	FeedId       int64 `json:"feed_id"`
	UnreadCount  int64 `json:"unread"`
	StarredCount int64 `json:"starred"`
}

func (s *Storage) FeedStats() ([]FeedStat, error) {
	rows, err := s.db.Query(fmt.Sprintf(`
		select
			feed_id,
			sum(case status when %d then 1 else 0 end),
			sum(case status when %d then 1 else 0 end)
		from items
		group by feed_id
	`, UNREAD, STARRED))
	if err != nil {
		log.Print(err)
		return nil, err
	}
	result := make([]FeedStat, 0)
	for rows.Next() {
		var stat FeedStat
		err = rows.Scan(&stat.FeedId, &stat.UnreadCount, &stat.StarredCount)
		if err != nil {
			log.Print(err)
			return nil, err
		}
		result = append(result, stat)
	}
	if err = rows.Err(); err != nil {
		log.Print(err)
		return nil, err
	}
	return result, nil
}

const (
	ITEMS_KEEP_SIZE = 50
	ITEMS_KEEP_DAYS = 90
)

// Delete old articles from the database to cleanup space.
//
// The rules:
//   - Never delete unread/starred entries.
//   - Keep at least the same amount of articles the feed provides (default: 50).
//     This prevents from deleting items for rarely updated and/or ever-growing
//     feeds which might eventually reappear as unread.
//   - Keep entries for a certain period (default: 90 days).
func (s *Storage) DeleteOldItems() {
	rows, err := s.db.Query(`
		select id, max(coalesce(size, 0), ?)
		from feeds
	`, ITEMS_KEEP_SIZE)
	if err != nil {
		log.Print(err)
		return
	}
	feedLimits := make(map[int64]int64)
	for rows.Next() {
		var feedId, limit int64
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
			time.Now().UTC().Add(-time.Duration(ITEMS_KEEP_DAYS)*24*time.Hour),
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
