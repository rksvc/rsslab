package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"rsslab/storage"
	"rsslab/utils"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/mmcdole/gofeed"
)

type Server struct {
	App atomic.Value

	db            *storage.Storage
	client        *http.Client
	pending       atomic.Int32
	lastRefreshed atomic.Value
	refresh       chan storage.Feed
	ticker        *time.Ticker
	context       context.Context
	cancel        context.CancelFunc
	mu            sync.Mutex
}

func New(db *storage.Storage) *Server {
	s := &Server{
		db:      db,
		client:  &http.Client{Timeout: 30 * time.Second},
		refresh: make(chan storage.Feed),
		context: context.Background(),
		cancel:  func() {},
	}
	for range 10 {
		go s.worker()
	}
	return s
}

func (s *Server) Start() {
	go func() {
		for {
			s.db.DeleteOldItems()
			s.db.PurgeCache()
			s.db.Vacuum()
			s.db.Optimize()
			time.Sleep(24 * time.Hour)
		}
	}()

	go s.FindFavicons()
	settings, err := s.db.GetSettings()
	if err != nil {
		log.Print(err)
		return
	}
	go s.SetRefreshRate(settings.RefreshRate)
	if settings.RefreshRate > 0 {
		go s.RefreshAllFeeds()
	}
}

func (s *Server) FindFavicons() {
	for _, feed := range s.db.ListFeedsMissingIcons() {
		s.FindFeedFavicon(feed)
	}
}

func (s *Server) FindFeedFavicon(feed storage.Feed) {
	var icon []byte
	for _, rawUrl := range []string{feed.Link, feed.FeedLink} {
		url, err := url.Parse(rawUrl)
		if err != nil || url.Host == "" {
			continue
		}
		resp, err := s.client.Get(fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", url.Host))
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if utils.IsErrorResponse(resp.StatusCode) {
			continue
		}
		icon, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Print(err)
			icon = nil
		}
		break
	}
	if len(icon) > 0 {
		s.db.UpdateFeedIcon(feed.Id, icon)
	}
}

func (s *Server) SetRefreshRate(minute int) {
	s.mu.Lock()
	s.cancel()
	s.context, s.cancel = context.WithCancel(context.Background())
	if minute <= 0 {
		if s.ticker != nil {
			s.ticker.Stop()
		}
		s.mu.Unlock()
		return
	}
	d := time.Duration(minute) * time.Minute
	if s.ticker != nil {
		s.ticker.Reset(d)
	} else {
		s.ticker = time.NewTicker(d)
	}
	s.mu.Unlock()

	log.Printf("auto-refresh %dm: starting", minute)
	for {
		select {
		case <-s.ticker.C:
			log.Printf("auto-refresh %dm: firing", minute)
			go s.RefreshAllFeeds()
		case <-s.context.Done():
			log.Printf("auto-refresh %dm: stopping", minute)
			return
		}
	}
}

func (s *Server) RefreshAllFeeds() {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		log.Print(err)
		return
	}
	s.lastRefreshed.Store(time.Now().UTC())
	s.RefreshFeeds(feeds...)
}

func (s *Server) RefreshFeeds(feeds ...storage.Feed) {
	log.Printf("refreshing %d feeds", len(feeds))
	s.pending.Add(int32(len(feeds)))
	for _, feed := range feeds {
		s.refresh <- feed
	}
}

func (s *Server) do(req *http.Request) (resp *http.Response, err error) {
	if req.URL.Scheme == "rsshub" {
		req.URL.RawPath = "/" + req.URL.Opaque
		req.URL.Opaque = ""
		req.URL.Path, err = url.PathUnescape(req.URL.RawPath)
		if err != nil {
			return
		}
		resp, err = s.App.Load().(*fiber.App).Test(req, -1)
	} else {
		req.Header.Add("User-Agent", utils.USER_AGENT)
		resp, err = s.client.Do(req)
	}
	if err == nil && utils.IsErrorResponse(resp.StatusCode) {
		resp.Body.Close()
		err = utils.ResponseError(resp)
	}
	return
}

func (s *Server) worker() {
	for feed := range s.refresh {
		items, state, err := s.listItems(feed)
		if err == nil {
			err = s.db.CreateItems(items, feed.Id, time.Now(), state)
		}
		if err != nil {
			log.Print(err)
		}
		s.db.SetFeedError(feed.Id, err)
		s.pending.Add(-1)
	}
}

func (s *Server) listItems(f storage.Feed) ([]storage.Item, *storage.HTTPState, error) {
	req, err := http.NewRequest(http.MethodGet, f.FeedLink, nil)
	if err != nil {
		return nil, nil, err
	}
	state, err := s.db.GetHTTPState(f.Id)
	if err != nil {
		return nil, nil, err
	}
	if state.LastModified != nil {
		req.Header.Set("If-Modified-Since", *state.LastModified)
	}
	if state.Etag != nil {
		req.Header.Set("If-None-Match", *state.Etag)
	}
	resp, err := s.do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil, nil
	}

	var body io.Reader = resp.Body
	if e := utils.GetEncoding(resp); e != nil {
		body = e.NewDecoder().Reader(body)
	}
	feed, err := gofeed.NewParser().Parse(body)
	if err != nil {
		return nil, nil, err
	}

	return convertItems(feed.Items, f), getHTTPState(resp), nil
}

func convertItems(items []*gofeed.Item, feed storage.Feed) []storage.Item {
	result := make([]storage.Item, len(items))
	now := time.Now()
	for i, item := range items {
		links := append([]string{item.Link}, item.Links...)
		link := utils.FirstNonEmpty(links...)
		if !utils.IsAPossibleLink(link) {
			link = utils.AbsoluteUrl(link, feed.Link)
		}
		result[i] = storage.Item{
			GUID:    utils.FirstNonEmpty(item.GUID, item.Link),
			FeedId:  feed.Id,
			Title:   item.Title,
			Link:    link,
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

func getHTTPState(resp *http.Response) *storage.HTTPState {
	lmod := resp.Header.Get("Last-Modified")
	etag := resp.Header.Get("Etag")
	if lmod != "" || etag != "" {
		return &storage.HTTPState{
			LastModified: &lmod,
			Etag:         &etag,
		}
	}
	return nil
}
