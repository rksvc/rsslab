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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/mmcdole/gofeed"
	"github.com/nkanaev/yarr/src/content/htmlutil"
	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
)

type Server struct {
	db     *storage.Storage
	client *resty.Client
	cache  *cache.Cache
	base   *url.URL

	pending atomic.Int32
	refresh *time.Ticker
	stopper chan struct{}
	mu      sync.Mutex
}

func New(db *storage.Storage, rsshubBaseUrl *url.URL) *Server {
	return &Server{
		db: db,
		client: resty.
			New().
			SetTimeout(30 * time.Second).
			SetDoNotParseResponse(true).
			OnBeforeRequest(func(c *resty.Client, r *resty.Request) error {
				url, err := url.Parse(r.URL)
				if err != nil {
					return err
				}
				if url.Scheme == "rsshub" {
					url.Scheme = rsshubBaseUrl.Scheme
					url.User = rsshubBaseUrl.User
					url.Host = rsshubBaseUrl.Host
					url.Path = rsshubBaseUrl.Path
					url = url.JoinPath(url.Opaque)
					url.Opaque = ""
					r.URL = url.String()
				}
				return nil
			}),
		cache: cache.NewCache(cache.NewLRU(), time.Hour),
		base:  rsshubBaseUrl,
	}
}

func (s *Server) Start() {
	go func() {
		s.db.DeleteOldItems()
		s.db.Vacuum()
		s.db.Optimize()
		for range time.NewTicker(24 * time.Hour).C {
			s.db.DeleteOldItems()
			s.db.Vacuum()
			s.db.Optimize()
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

	for i := 0; i < NUM_WORKERS; i++ {
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

func (s *Server) worker(srcQueue <-chan storage.Feed, dstQueue chan<- []storage.Item) {
	for feed := range srcQueue {
		items, err := s.listItems(feed)
		if err != nil {
			s.db.SetFeedError(feed.Id, err)
		} else {
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

	req := s.client.R()
	if state.LastModified != nil {
		req.SetHeader("If-Modified-Since", *state.LastModified)
	}
	if state.Etag != nil {
		req.SetHeader("If-None-Match", *state.Etag)
	}
	resp, err := req.Get(f.FeedLink)
	if err != nil {
		return nil, err
	}
	rawBody := resp.RawBody()
	defer rawBody.Close()
	switch {
	case resp.StatusCode() == http.StatusNotModified:
		return nil, nil
	case resp.StatusCode() < 200 || resp.StatusCode() >= 300:
		return nil, fmt.Errorf(`%s "%s": %s`, resp.Request.Method, resp.Request.URL, resp.Status())
	}

	var body io.Reader = rawBody
	if e := getEncoding(resp); e != nil {
		body = e.NewDecoder().Reader(body)
	}
	feed, err := gofeed.NewParser().Parse(body)
	if err != nil {
		return nil, err
	}

	lmod := resp.Header().Get("Last-Modified")
	etag := resp.Header().Get("Etag")
	if lmod != "" || etag != "" {
		err = s.db.SetHTTPState(f.Id, lmod, etag)
		if err != nil {
			return nil, err
		}
	}
	return convertItems(feed.Items, f), nil
}

var imageTypes = map[string]struct{}{
	"image/x-icon": {},
	"image/png":    {},
	"image/jpeg":   {},
	"image/gif":    {},
}

func (s *Server) findFavicon(siteUrl, feedUrl string) *[]byte {
	favicon := func(link string) string {
		url, err := url.Parse(link)
		if err != nil {
			return ""
		}
		return fmt.Sprintf("%s://%s/favicon.ico", url.Scheme, url.Host)
	}

	var icons []string
	if siteUrl != "" {
		if resp, err := s.client.
			R().
			SetHeader("User-Agent", utils.UserAgent).
			SetHeader("Accept-Language", utils.AcceptLanguage).
			Get(siteUrl); err == nil {
			body := resp.RawBody()
			doc, err := goquery.NewDocumentFromReader(body)
			body.Close()
			if err == nil {
				doc.Find("link[rel=icon]").Each(func(_ int, s *goquery.Selection) {
					if val, exists := s.Attr("href"); exists {
						if icon := htmlutil.AbsoluteUrl(val, siteUrl); icon != "" {
							icons = append(icons, icon)
						}
					}
				})
			}
		}
		if icon := favicon(siteUrl); icon != "" {
			icons = append(icons, icon)
		}
	}
	if icon := favicon(feedUrl); icon != "" {
		icons = append(icons, icon)
	}

	for _, icon := range icons {
		resp, err := s.client.
			R().
			SetHeader("User-Agent", utils.UserAgent).
			SetHeader("Accept-Language", utils.AcceptLanguage).
			Get(icon)
		if err != nil {
			continue
		}
		body := resp.RawBody()
		content, err := io.ReadAll(body)
		body.Close()
		if err != nil {
			continue
		} else if _, ok := imageTypes[http.DetectContentType(content)]; ok {
			return &content
		}
	}
	return nil
}

func convertItems(items []*gofeed.Item, feed storage.Feed) []storage.Item {
	result := make([]storage.Item, len(items))
	now := time.Now()
	for i, item := range items {
		links := []string{item.Link}
		links = append(links, item.Links...)
		result[i] = storage.Item{
			GUID:    firstNonEmpty(item.GUID, item.Link),
			FeedId:  feed.Id,
			Title:   item.Title,
			Link:    firstNonEmpty(links...),
			Content: firstNonEmpty(item.Content, item.Description),
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

func getEncoding(resp *resty.Response) encoding.Encoding {
	contentType := resp.Header().Get("Content-Type")
	if _, params, err := mime.ParseMediaType(contentType); err == nil {
		if cs, ok := params["charset"]; ok {
			e, _ := charset.Lookup(cs)
			return e
		}
	}
	return nil
}

func firstNonEmpty(vals ...string) string {
	for _, val := range vals {
		if val = strings.TrimSpace(val); val != "" {
			return val
		}
	}
	return ""
}
