package server

import (
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"rsslab/cache"
	"rsslab/storage"
	"rsslab/utils"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/mmcdole/gofeed"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
)

type Server struct {
	App atomic.Value

	db       *storage.Storage
	client   *http.Client
	cacheTTL time.Duration
	cache    *cache.Cache

	pending atomic.Int32
	refresh *time.Ticker
	stopper chan struct{}
	mu      sync.Mutex
}

func New(db *storage.Storage) *Server {
	return &Server{
		db:       db,
		client:   &http.Client{Timeout: 30 * time.Second},
		cacheTTL: time.Hour,
		cache:    cache.NewCache(cache.NewLRU()),
	}
}

func (s *Server) Start() {
	go func() {
		for {
			s.db.DeleteOldItems()
			s.db.Vacuum()
			s.db.Optimize()
			time.Sleep(24 * time.Hour)
		}
	}()

	refreshRate, _ := s.db.GetSettingsValueInt64("refresh_rate")
	go s.FindFavicons()
	go s.SetRefreshRate(refreshRate)
	if refreshRate > 0 {
		go s.RefreshAllFeeds()
	}
}

const NUM_WORKERS = 7

func (s *Server) FindFavicons() {
	for _, feed := range s.db.ListFeedsMissingIcons() {
		s.FindFeedFavicon(feed)
	}
}

func (s *Server) FindFeedFavicon(feed storage.Feed) {
	icon := s.findFavicon(feed.Link, feed.FeedLink)
	if icon == nil {
		return
	}
	s.db.UpdateFeedIcon(feed.Id, icon)
}

func (s *Server) SetRefreshRate(minute int64) {
	if s.stopper != nil {
		s.refresh.Stop()
		s.refresh = nil
		s.stopper <- struct{}{}
		s.stopper = nil
	}
	if minute == 0 {
		return
	}

	s.stopper = make(chan struct{})
	s.refresh = time.NewTicker(time.Duration(minute) * time.Minute)

	log.Printf("auto-refresh %dm: starting", minute)
	for {
		select {
		case <-s.refresh.C:
			log.Printf("auto-refresh %dm: firing", minute)
			go s.RefreshAllFeeds()
		case <-s.stopper:
			log.Printf("auto-refresh %dm: stopping", minute)
			return
		}
	}
}

func (s *Server) RefreshAllFeeds() {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return
	}
	s.RefreshFeeds(feeds...)
}

func (s *Server) RefreshFeeds(feeds ...storage.Feed) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Print("refreshing feeds")
	s.pending.Store(int32(len(feeds)))
	s.refresher(feeds)
}

func (s *Server) refresher(feeds []storage.Feed) {
	srcQueue := make(chan storage.Feed)
	dstQueue := make(chan []storage.Item)

	for range NUM_WORKERS {
		go s.worker(srcQueue, dstQueue)
	}
	go func() {
		for _, feed := range feeds {
			srcQueue <- feed
		}
	}()

	for range feeds {
		items := <-dstQueue
		if len(items) > 0 {
			if s.db.CreateItems(items) == nil {
				s.db.SetFeedSize(items[0].FeedId, len(items))
			}
		}
		s.pending.Add(-1)
		s.db.SyncSearch()
	}
	close(srcQueue)
	close(dstQueue)

	log.Printf("finished refreshing %d feeds", len(feeds))
}

func (s *Server) do(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "rsshub" {
		req.URL.Path = "/" + req.URL.Opaque
		req.URL.Opaque = ""
		return s.App.Load().(*fiber.App).Test(req, -1)
	}
	return s.client.Do(req)
}

func (s *Server) worker(srcQueue <-chan storage.Feed, dstQueue chan<- []storage.Item) {
	for feed := range srcQueue {
		items, err := s.listItems(feed)
		if err != nil {
			s.db.SetFeedError(feed.Id, err)
		} else {
			s.db.SetFeedLastRefreshed(feed.Id, time.Now())
			s.db.ResetFeedError(feed.Id)
		}
		dstQueue <- items
	}
}

func (s *Server) listItems(f storage.Feed) ([]storage.Item, error) {
	state, err := s.db.GetHTTPState(f.Id)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", f.FeedLink, nil)
	if err != nil {
		return nil, err
	}
	if state.LastModified != nil {
		req.Header.Set("If-Modified-Since", *state.LastModified)
	}
	if state.Etag != nil {
		req.Header.Set("If-None-Match", *state.Etag)
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	} else if utils.IsErrorResponse(resp.StatusCode) {
		return nil, fmt.Errorf(`%s "%s": %s`, resp.Request.Method, resp.Request.URL, resp.Status)
	}

	var body io.Reader = resp.Body
	if e := getEncoding(resp); e != nil {
		body = e.NewDecoder().Reader(body)
	}
	feed, err := gofeed.NewParser().Parse(body)
	if err != nil {
		return nil, err
	}

	lmod := resp.Header.Get("Last-Modified")
	etag := resp.Header.Get("Etag")
	if lmod != "" || etag != "" {
		err = s.db.SetHTTPState(f.Id, lmod, etag)
		if err != nil {
			return nil, err
		}
	}
	return convertItems(feed.Items, f), nil
}

func (s *Server) findFavicon(siteUrl, feedUrl string) *[]byte {
	for _, rawUrl := range []string{siteUrl, feedUrl} {
		url, err := url.Parse(rawUrl)
		if err != nil || url.Host == "" {
			continue
		}
		resp, err := s.client.Get(fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", url.Host))
		if err != nil || utils.IsErrorResponse(resp.StatusCode) {
			continue
		}
		icon, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil
		}
		return &icon
	}
	return nil
}

func convertItems(items []*gofeed.Item, feed storage.Feed) []storage.Item {
	result := make([]storage.Item, len(items))
	now := time.Now()
	for i, item := range items {
		links := append([]string{item.Link}, item.Links...)
		result[i] = storage.Item{
			GUID:    utils.FirstNonEmpty(item.GUID, item.Link),
			FeedId:  feed.Id,
			Title:   item.Title,
			Link:    utils.FirstNonEmpty(links...),
			Content: utils.FirstNonEmpty(item.Content, item.Description),
			Status:  storage.UNREAD,
		}
		if item.PublishedParsed != nil {
			result[i].Date = *item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			result[i].Date = *item.UpdatedParsed
		} else {
			result[i].Date = now
		}
		if item.Image != nil {
			result[i].ImageURL = &item.Image.URL
		}
	}
	return result
}

func getEncoding(resp *http.Response) encoding.Encoding {
	contentType := resp.Header.Get("Content-Type")
	if _, params, err := mime.ParseMediaType(contentType); err == nil {
		if cs, ok := params["charset"]; ok {
			e, _ := charset.Lookup(cs)
			return e
		}
	}
	return nil
}
