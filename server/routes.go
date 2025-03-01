package server

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path"
	"rsslab/rss"
	"rsslab/storage"
	"rsslab/utils"
	"strings"
	"time"
)

type dict = map[string]any

type errBadRequest struct {
	Err error
}

func (err *errBadRequest) Error() string {
	return err.Err.Error()
}

type gzipResponseWriter struct {
	out *gzip.Writer
	src http.ResponseWriter
}

func (gz *gzipResponseWriter) Header() http.Header {
	return gz.src.Header()
}

func (gz *gzipResponseWriter) Write(p []byte) (int, error) {
	return gz.out.Write(p)
}

func (gz *gzipResponseWriter) WriteHeader(statusCode int) {
	gz.src.WriteHeader(statusCode)
}

func wrap(handleFunc func(context) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			gz := &gzipResponseWriter{out: gzip.NewWriter(w), src: w}
			defer func() {
				if err := gz.out.Close(); err != nil {
					log.Print(err)
				}
			}()
			w = gz
		}
		if err := handleFunc(context{w, r}); err != nil {
			log.Printf("%s %s: %s", r.Method, r.URL.EscapedPath(), err)
			if _, ok := err.(*errBadRequest); ok {
				w.WriteHeader(http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			_, err = w.Write(utils.StringToBytes(err.Error()))
			if err != nil {
				log.Print(err)
			}
		}
	}
}

//go:embed dist
var assets embed.FS

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimLeft(r.URL.Path, "/")
	if p == "" {
		p = "index.html"
	}
	if p == "index.html" {
		if dark, err := s.db.GetSettingInt(storage.DARK_THEME); err == nil {
			var new []byte
			if dark != nil && *dark != 0 {
				new = []byte{'1'}
			}
			b, _ := assets.ReadFile(path.Join("dist", p))
			w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(p)))
			w.Write(bytes.Replace(b, []byte("%DARK_THEME%"), new, 1))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write(utils.StringToBytes(err.Error()))
		}
	} else if f, err := assets.Open(path.Join("dist", p+".gz")); err == nil {
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(p)))
		w.Header().Set("Cache-Control", "max-age=31536000") // 1 year
		io.Copy(w, f)
	} else if errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, r)
	} else {
		log.Print(err)
	}
}

func (s *Server) handleStatus(c context) error {
	state, err := s.db.FeedState()
	if err != nil {
		return err
	}
	return c.JSON(dict{
		"state":          state,
		"running":        s.pending.Load(),
		"last_refreshed": s.lastRefreshed.Load(),
	})
}

func (s *Server) handleFolderList(c context) error {
	folders, err := s.db.ListFolders()
	if err != nil {
		return err
	}
	return c.JSON(folders)
}

func (s *Server) handleFolderCreate(c context) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := c.ParseBody(&body); err != nil {
		return err
	}
	folder, err := s.db.CreateFolder(body.Title)
	if err != nil {
		return err
	}
	return c.JSON(folder)
}

func (s *Server) handleFolderUpdate(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	var editor storage.FolderEditor
	if err = c.ParseBody(&editor); err != nil {
		return err
	}
	return s.db.EditFolder(id, editor)
}

func (s *Server) handleFolderDelete(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	return s.db.DeleteFolder(id)
}

func (s *Server) handleFolderRefresh(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	feeds, err := s.db.GetFeeds(id)
	if err != nil {
		return err
	}
	go s.RefreshFeeds(feeds...)
	return nil
}

func (s *Server) handleFeedList(c context) error {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return err
	}
	return c.JSON(feeds)
}

func (s *Server) handleFeedCreate(c context) error {
	var body struct {
		Url      string `json:"url"`
		FolderId *int   `json:"folder_id"`
	}
	if err := c.ParseBody(&body); err != nil {
		return err
	}

	var state storage.HTTPState
	rawFeed, err := s.do(body.Url, &state)
	if err != nil {
		return err
	}
	feed, err := s.db.CreateFeed(
		rawFeed.Title,
		rawFeed.SiteURL,
		body.Url,
		body.FolderId,
	)
	if err != nil {
		return err
	}

	items := convertItems(rawFeed.Items, *feed)
	lastRefreshed := time.Now()
	if err = s.db.CreateItems(items, feed.Id, lastRefreshed, &state); err != nil {
		return err
	}

	feed.LastRefreshed = &lastRefreshed
	return c.JSON(dict{
		"feed":       feed,
		"item_count": len(items),
	})
}

func (s *Server) handleFeedsRefresh(c context) error {
	go s.RefreshAllFeeds()
	return nil
}

func (s *Server) handleFeedRefresh(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	feed, err := s.db.GetFeed(id)
	if err != nil {
		return err
	} else if feed == nil {
		return c.NotFound()
	}
	go s.RefreshFeeds(*feed)
	return nil
}

func (s *Server) handleFeedUpdate(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	var body struct {
		Title    *string `json:"title"`
		FeedLink *string `json:"feed_link"`
		FolderId *int    `json:"folder_id"`
	}
	if err = c.ParseBody(&body); err != nil {
		return err
	}
	editor := storage.FeedEditor{
		Title:    body.Title,
		FeedLink: body.FeedLink,
	}
	if body.FolderId != nil {
		if *body.FolderId < 0 {
			body.FolderId = nil
		}
		editor.FolderId = &body.FolderId
	}
	return s.db.EditFeed(id, editor)
}

func (s *Server) handleFeedDelete(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	return s.db.DeleteFeed(id)
}

func (s *Server) handleItemList(c context) error {
	var filter storage.ItemFilter
	if err := c.ParseQuery(&filter); err != nil {
		return err
	}

	const PER_PAGE = 20
	items, err := s.db.ListItems(filter, PER_PAGE+1)
	if err != nil {
		return err
	}
	hasMore := false
	if len(items) > PER_PAGE {
		hasMore = true
		items = items[:PER_PAGE]
	}
	return c.JSON(dict{
		"list":     items,
		"has_more": hasMore,
	})
}

func (s *Server) handleItemRead(c context) error {
	var filter storage.ItemFilter
	if err := c.ParseQuery(&filter); err != nil {
		return err
	}
	return s.db.MarkItemsRead(filter)
}

func (s *Server) handleItem(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	item, err := s.db.GetItem(id)
	if err != nil {
		return err
	} else if item == nil {
		return c.NotFound()
	}
	item.Content = utils.Sanitize(item.Link, item.Content)
	return c.JSON(item)
}

func (s *Server) handleItemUpdate(c context) error {
	id, err := c.VarInt("id")
	if err != nil {
		return err
	}
	var body struct {
		Status storage.ItemStatus `json:"status"`
	}
	if err = c.ParseBody(&body); err != nil {
		return err
	}
	return s.db.UpdateItemStatus(id, body.Status)
}

func (s *Server) handleSettings(c context) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return err
	}
	return c.JSON(settings)
}

func (s *Server) handleSettingsUpdate(c context) error {
	var settings map[string]any
	if err := c.ParseBody(&settings); err != nil {
		return err
	}
	for key, val := range settings {
		if err := s.db.UpdateSetting(key, val); err != nil {
			return err
		}
		if key == storage.REFRESH_RATE {
			go s.SetRefreshRate(int(val.(float64)))
		}
	}
	return nil
}

func (s *Server) handleOPMLImport(c context) error {
	_, fh, err := c.r.FormFile("opml")
	if err != nil {
		return &errBadRequest{err}
	}
	file, err := fh.Open()
	if err != nil {
		return &errBadRequest{err}
	}
	d := utils.XMLDecoder(file)
	d.Entity = xml.HTMLEntity
	var opml rss.OPML
	err = d.Decode(&opml)
	if err != nil {
		return &errBadRequest{err}
	}

	var errs []error
	for _, o := range opml.Outlines {
		if o.IsFolder() {
			title := o.Title
			if title == "" {
				title = o.Title2
			}
			folder, err := s.db.CreateFolder(title)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, o := range o.AllFeeds() {
				_, err = s.db.CreateFeed(o.Title, o.SiteUrl, o.FeedUrl, &folder.Id)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			}
		} else {
			_, err := s.db.CreateFeed(o.Title, o.SiteUrl, o.FeedUrl, nil)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	go s.RefreshAllFeeds()
	return nil
}

func (s *Server) handleOPMLExport(c context) error {
	opml := rss.OPML{
		Version: "1.1",
		Title:   "subscriptions",
	}

	feedsByFolderId := make(map[int][]*storage.Feed)
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		if feed.FolderId == nil {
			opml.Outlines = append(opml.Outlines, rss.Outline{
				Type:    "rss",
				Title:   feed.Title,
				FeedUrl: feed.FeedLink,
				SiteUrl: feed.Link,
			})
		} else {
			id := *feed.FolderId
			feedsByFolderId[id] = append(feedsByFolderId[id], &feed)
		}
	}

	folders, err := s.db.ListFolders()
	if err != nil {
		return err
	}
	for _, folder := range folders {
		feeds := feedsByFolderId[folder.Id]
		if len(feeds) == 0 {
			continue
		}
		folder := rss.Outline{Title: folder.Title}
		for _, feed := range feeds {
			folder.Outlines = append(folder.Outlines, rss.Outline{
				Type:    "rss",
				Title:   feed.Title,
				FeedUrl: feed.FeedLink,
				SiteUrl: feed.Link,
			})
		}
		opml.Outlines = append(opml.Outlines, folder)
	}

	c.w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.w.Header().Set("Content-Disposition", `attachment; filename="subscriptions.opml"`)
	c.Write(utils.StringToBytes(xml.Header))
	e := xml.NewEncoder(c.w)
	e.Indent("", "  ")
	return e.Encode(opml)
}

func (s *Server) handleTransform(c context) error {
	typ := c.r.PathValue("type")
	params := c.r.PathValue("params")
	var state storage.HTTPState
	feed, err := s.do(typ+":"+params, &state)
	if err != nil {
		return err
	}
	c.w.Header().Set("Content-Type", "application/feed+json; charset=UTF-8")
	return json.NewEncoder(c.w).Encode(feed)
}
