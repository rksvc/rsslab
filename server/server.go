package server

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"rsslab/rss"
	"rsslab/storage"
	"rsslab/utils"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/net/html/charset"
	"golang.org/x/net/publicsuffix"
)

type Server struct {
	URL           string
	db            *storage.Storage
	client        http.Client
	pending       atomic.Int32
	lastRefreshed atomic.Value
	refresh       chan storage.Feed
	ticker        *time.Ticker
	cancel        chan struct{}
	mu            sync.Mutex
}

func New(db *storage.Storage) *Server {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		panic(err)
	}
	s := &Server{
		db: db,
		client: http.Client{
			Timeout: 30 * time.Second,
			Jar:     jar,
		},
		refresh: make(chan storage.Feed),
		cancel:  make(chan struct{}),
	}

	go func() {
		for {
			s.db.DeleteOldItems()
			s.db.Vacuum()
			s.db.Optimize()
			time.Sleep(24 * time.Hour)
		}
	}()
	for range 10 {
		go s.worker()
	}
	go s.FindFavicons()

	refreshRate, err := s.db.GetSettingInt(storage.REFRESH_RATE)
	if err != nil {
		log.Print(err)
	} else if refreshRate != nil {
		go s.SetRefreshRate(*refreshRate)
		if *refreshRate > 0 {
			go s.RefreshAllFeeds()
		}
	}

	return s
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET    /api/status", wrap(s.handleStatus))
	mux.HandleFunc("GET    /api/folders", wrap(s.handleFolderList))
	mux.HandleFunc("POST   /api/folders", wrap(s.handleFolderCreate))
	mux.HandleFunc("PUT    /api/folders/{id}", wrap(s.handleFolderUpdate))
	mux.HandleFunc("DELETE /api/folders/{id}", wrap(s.handleFolderDelete))
	mux.HandleFunc("POST   /api/folders/{id}/refresh", wrap(s.handleFolderRefresh))
	mux.HandleFunc("GET    /api/feeds", wrap(s.handleFeedList))
	mux.HandleFunc("POST   /api/feeds", wrap(s.handleFeedCreate))
	mux.HandleFunc("POST   /api/feeds/refresh", wrap(s.handleFeedsRefresh))
	mux.HandleFunc("GET    /api/feeds/{id}/icon", wrap(s.handleFeedIcon))
	mux.HandleFunc("POST   /api/feeds/{id}/refresh", wrap(s.handleFeedRefresh))
	mux.HandleFunc("PUT    /api/feeds/{id}", wrap(s.handleFeedUpdate))
	mux.HandleFunc("DELETE /api/feeds/{id}", wrap(s.handleFeedDelete))
	mux.HandleFunc("GET    /api/items", wrap(s.handleItemList))
	mux.HandleFunc("PUT    /api/items", wrap(s.handleItemRead))
	mux.HandleFunc("GET    /api/items/{id}", wrap(s.handleItem))
	mux.HandleFunc("PUT    /api/items/{id}", wrap(s.handleItemUpdate))
	mux.HandleFunc("GET    /api/settings", wrap(s.handleSettings))
	mux.HandleFunc("PUT    /api/settings", wrap(s.handleSettingsUpdate))
	mux.HandleFunc("POST   /api/opml/import", wrap(s.handleOPMLImport))
	mux.HandleFunc("GET    /api/opml/export", wrap(s.handleOPMLExport))
	mux.HandleFunc("GET    /api/transform/{type}", wrap(s.handleTransform))
	mux.HandleFunc("GET    /", s.handleIndex)

	host, port := addr, ""
	if i := strings.LastIndexByte(addr, ':'); i != -1 {
		host, port = addr[:i], addr[i+1:]
	}
	if host == "" {
		host = "0.0.0.0"
	}
	s.URL = fmt.Sprintf("http://%s:%s", host, port)
	log.Print("server started on " + s.URL)
	return (&http.Server{
		Addr:    addr,
		Handler: mux,
	}).ListenAndServe()
}

func (s *Server) FindFavicons() {
	for _, feed := range s.db.ListFeedsMissingIcons() {
		s.FindFeedFavicon(feed)
	}
}

func (s *Server) FindFeedFavicon(feed storage.Feed) {
	for _, rawUrl := range []string{feed.Link, feed.FeedLink} {
		url, err := url.Parse(rawUrl)
		if err != nil || url.Host == "" {
			continue
		}
		resp, err := s.client.Get(fmt.Sprintf("https://icons.duckduckgo.com/ip3/%s.ico", url.Host))
		if err != nil {
			log.Print(err)
			continue
		}
		defer resp.Body.Close()
		if utils.IsErrorResponse(resp.StatusCode) {
			if resp.StatusCode != http.StatusNotFound {
				log.Print(utils.ResponseError(resp))
			}
			continue
		}
		icon, err := io.ReadAll(resp.Body)
		if err == nil {
			s.db.UpdateFeedIcon(feed.Id, icon)
		} else {
			log.Print(err)
		}
		break
	}
}

func (s *Server) SetRefreshRate(minute int) {
	s.mu.Lock()
	close(s.cancel)
	s.cancel = make(chan struct{})
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
	cancel := s.cancel
	s.mu.Unlock()

	log.Printf("auto-refresh %dm: starting", minute)
	for {
		select {
		case <-s.ticker.C:
			log.Printf("auto-refresh %dm: firing", minute)
			go s.RefreshAllFeeds()
		case <-cancel:
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

func (s *Server) do(rawUrl string, state *storage.HTTPState) (*rss.Feed, error) {
	url, err := url.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "rsslab" {
		switch url.Host {
		case "html":
			rule := new(rss.HTMLRule)
			if err := utils.ParseQuery(url, rule); err != nil {
				return nil, err
			}
			return rss.TransformHTML(rule, &s.client)

		case "json":
			rule := new(rss.JSONRule)
			if err := utils.ParseQuery(url, rule); err != nil {
				return nil, err
			}
			return rss.TransformJSON(rule, &s.client)

		case "js":
			rule := new(rss.JavaScriptRule)
			if err := utils.ParseQuery(url, rule); err != nil {
				return nil, err
			}
			return rss.RunJavaScript(rule, &s.client)

		default:
			return nil, errors.New("invalid URL")
		}
	}

	req, err := http.NewRequest(http.MethodGet, rawUrl, nil)
	if err != nil {
		return nil, err
	}
	if state != nil {
		if state.LastModified != nil {
			req.Header.Set("If-Modified-Since", *state.LastModified)
		}
		if state.Etag != nil {
			req.Header.Set("If-None-Match", *state.Etag)
		}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Miniflux/dev; +https://miniflux.app)")

	resp, err := s.client.Do(req)
	if err == nil && utils.IsErrorResponse(resp.StatusCode) {
		resp.Body.Close()
		err = utils.ResponseError(resp)
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}

	lmod := resp.Header.Get("Last-Modified")
	etag := resp.Header.Get("Etag")
	if lmod != "" || etag != "" {
		state.LastModified = &lmod
		state.Etag = &etag
	}

	var b io.Reader = resp.Body
	if _, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type")); err == nil {
		if cs, ok := params["charset"]; ok {
			if e, _ := charset.Lookup(cs); e != nil {
				b = e.NewDecoder().Reader(b)
			}
		}
	}
	return rss.Parse(b, rawUrl)
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
	state, err := s.db.GetHTTPState(f.Id)
	if err != nil {
		return nil, nil, err
	}
	feed, err := s.do(f.FeedLink, &state)
	if err != nil || feed == nil {
		return nil, nil, err
	}
	return convertItems(feed.Items, f), &state, nil
}

func convertItems(items []rss.Item, feed storage.Feed) []storage.Item {
	result := make([]storage.Item, len(items))
	now := time.Now()
	for i, item := range items {
		result[i] = storage.Item{
			GUID:    cmp.Or(item.GUID, item.URL),
			FeedId:  feed.Id,
			Title:   item.Title,
			Link:    item.URL,
			Content: item.Content,
			Status:  storage.UNREAD,
		}
		if item.Date == nil {
			result[i].Date = now
		} else {
			result[i].Date = *item.Date
		}
		if item.ImageURL != "" {
			result[i].ImageURL = &item.ImageURL
		}
	}
	return result
}
