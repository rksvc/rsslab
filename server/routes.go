package server

import (
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"rsslab/storage"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/mmcdole/gofeed"
	"github.com/nkanaev/yarr/src/content/htmlutil"
	"github.com/nkanaev/yarr/src/content/sanitizer"
	"github.com/nkanaev/yarr/src/server/opml"
)

func (s *Server) Register(api fiber.Router) {
	api.Get("/status", s.handleStatus)
	api.Get("/folders", s.handleFolderList)
	api.Post("/folders", s.handleFolderCreate)
	api.Put("/folders/:id", s.handleFolderUpdate)
	api.Delete("/folders/:id", s.handleFolderDelete)
	api.Post("/folders/:id/refresh", s.handleFolderRefresh)
	api.Get("/feeds", s.handleFeedList)
	api.Post("/feeds", s.handleFeedCreate)
	api.Post("/feeds/refresh", s.handleFeedsRefresh)
	api.Get("/feeds/errors", s.handleFeedErrorsList)
	api.Post("/feeds/errors/refresh", s.handleErrorsRefresh)
	api.Get("/feeds/:id/icon", s.handleFeedIcon)
	api.Post("/feeds/:id/refresh", s.handleFeedRefresh)
	api.Put("/feeds/:id", s.handleFeedUpdate)
	api.Delete("/feeds/:id", s.handleFeedDelete)
	api.Get("/items", s.handleItemList)
	api.Put("/items", s.handleItemRead)
	api.Get("/items/:id", s.handleItem)
	api.Put("/items/:id", s.handleItemUpdate)
	api.Get("/settings", s.handleSettings)
	api.Put("/settings", s.handleSettingsUpdate)
	api.Post("/opml/import", s.handleOPMLImport)
	api.Get("/opml/export", s.handleOPMLExport)
}

func (s *Server) handleStatus(c fiber.Ctx) error {
	stats, err := s.db.FeedStats()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(fiber.Map{
		"stats":   stats,
		"running": s.pending.Load(),
	})
}

func (s *Server) handleFolderList(c fiber.Ctx) error {
	folders, err := s.db.ListFolders()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(folders)
}

func (s *Server) handleFolderCreate(c fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	folder, err := s.db.CreateFolder(body.Title)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.Status(http.StatusCreated).JSON(folder)
}

func (s *Server) handleFolderUpdate(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body struct {
		Title      *string `json:"title"`
		IsExpanded *bool   `json:"is_expanded"`
	}
	if err = c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if body.Title != nil {
		if err = s.db.RenameFolder(id, *body.Title); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
	}
	if body.IsExpanded != nil {
		if err = s.db.ToggleFolderExpanded(id, *body.IsExpanded); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderDelete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err = s.db.DeleteFolder(id); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderRefresh(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	feeds, err := s.db.GetFeeds(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	go s.RefreshFeeds(feeds...)
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedList(c fiber.Ctx) error {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(feeds)
}

func (s *Server) handleFeedCreate(c fiber.Ctx) error {
	var body struct {
		Url      string `json:"url"`
		FolderId *int64 `json:"folder_id"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	resp, err := s.client.R().Get(body.Url)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	rawBody := resp.RawBody()
	defer rawBody.Close()
	var f io.Reader = rawBody
	if e := getEncoding(resp); e != nil {
		f = e.NewDecoder().Reader(f)
	}
	rawFeed, err := gofeed.NewParser().Parse(f)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	rawFeed.FeedLink = body.Url
	feed, err := s.db.CreateFeed(
		rawFeed.Title,
		rawFeed.Description,
		rawFeed.Link,
		rawFeed.FeedLink,
		body.FolderId,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	items := convertItems(rawFeed.Items, *feed)
	if len(items) > 0 {
		if err = s.db.CreateItems(items); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		if err = s.db.SetFeedSize(feed.Id, len(items)); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		if err = s.db.SyncSearch(); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
	}
	go s.FindFeedFavicon(*feed)
	lastRefreshed := time.Now()
	if err = s.db.SetFeedLastRefreshed(feed.Id, lastRefreshed); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}

	feed.LastRefreshed = &lastRefreshed
	return c.JSON(feed)
}

func (s *Server) handleFeedsRefresh(c fiber.Ctx) error {
	go s.RefreshAllFeeds()
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedErrorsList(c fiber.Ctx) error {
	errors, err := s.db.GetFeedErrors()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(errors)
}

func (s *Server) handleErrorsRefresh(c fiber.Ctx) error {
	feeds, err := s.db.GetFeedsWithErrors()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	go s.RefreshFeeds(feeds...)
	return c.SendStatus(http.StatusOK)
}

type icon struct {
	ctype string
	bytes []byte
	etag  string
}

func (s *Server) handleFeedIcon(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	dat, err := s.cache.TryGet(strconv.FormatInt(id, 10), true, func() (any, error) {
		feed, err := s.db.GetFeed(id)
		if err != nil {
			return nil, err
		} else if feed == nil || feed.Icon == nil {
			return nil, nil
		}
		return &icon{
			ctype: http.DetectContentType(*feed.Icon),
			bytes: *feed.Icon,
			etag:  fmt.Sprintf("%x", md5.Sum(*feed.Icon)),
		}, nil
	})
	if err != nil {
		log.Print(err)
		return c.SendStatus(http.StatusInternalServerError)
	} else if dat == nil {
		return c.SendStatus(http.StatusNotFound)
	}

	icon := dat.(*icon)
	if string(c.Request().Header.Peek("If-None-Match")) == icon.etag {
		return c.SendStatus(http.StatusNotModified)
	}
	c.Response().Header.SetContentType(icon.ctype)
	c.Response().Header.Set("Etag", icon.etag)
	_, err = c.Write(icon.bytes)
	return err
}

func (s *Server) handleFeedRefresh(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	feed, err := s.db.GetFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	} else if feed == nil {
		return c.Status(http.StatusBadRequest).SendString("no such feed")
	}
	go s.RefreshFeeds(*feed)
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedUpdate(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body map[string]any
	if err = c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if title, ok := body["title"]; ok {
		if title, ok := title.(string); ok {
			if err = s.db.RenameFeed(id, title); err != nil {
				return c.Status(http.StatusInternalServerError).SendString(err.Error())
			}
		}
	}
	if feedLink, ok := body["feed_link"]; ok {
		if feedLink, ok := feedLink.(string); ok {
			if err = s.db.EditFeedLink(id, feedLink); err != nil {
				return c.Status(http.StatusInternalServerError).SendString(err.Error())
			}
		}
	}
	if folderId, ok := body["folder_id"]; ok {
		if folderId == nil {
			err = s.db.UpdateFeedFolder(id, nil)
		} else if folderId, ok := folderId.(float64); ok {
			folderId := int64(folderId)
			err = s.db.UpdateFeedFolder(id, &folderId)
		}
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedDelete(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err = s.db.DeleteFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItemList(c fiber.Ctx) error {
	var filter storage.ItemFilter
	if folderId := c.Query("folder_id"); folderId != "" {
		folderId, err := strconv.ParseInt(folderId, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}
		filter.FolderId = &folderId
	}
	if feedId := c.Query("feed_id"); feedId != "" {
		feedId, err := strconv.ParseInt(feedId, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}
		filter.FeedId = &feedId
	}
	if after := c.Query("after"); after != "" {
		after, err := strconv.ParseInt(after, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}
		filter.After = &after
	}
	if status := c.Query("status"); status != "" {
		status := storage.StatusValues[status]
		filter.Status = &status
	}
	if search := c.Query("search"); search != "" {
		filter.Search = &search
	}
	newestFirst := c.Query("oldest_first") != "true"

	const PER_PAGE = 20
	items, err := s.db.ListItems(filter, PER_PAGE+1, newestFirst)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	hasMore := false
	if len(items) == PER_PAGE+1 {
		hasMore = true
		items = items[:PER_PAGE]
	}
	return c.JSON(fiber.Map{
		"list":     items,
		"has_more": hasMore,
	})
}

func (s *Server) handleItemRead(c fiber.Ctx) error {
	var filter storage.MarkFilter
	if folderId := c.Query("folder_id"); folderId != "" {
		folderId, err := strconv.ParseInt(folderId, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}
		filter.FolderId = &folderId
	}
	if feedId := c.Query("feed_id"); feedId != "" {
		feedId, err := strconv.ParseInt(feedId, 10, 64)
		if err != nil {
			return c.Status(http.StatusBadRequest).SendString(err.Error())
		}
		filter.FeedId = &feedId
	}
	err := s.db.MarkItemsRead(filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItem(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	item, err := s.db.GetItem(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	} else if item == nil {
		return c.Status(http.StatusBadRequest).SendString("no such item")
	}

	// runtime fix for relative links
	if !htmlutil.IsAPossibleLink(item.Link) {
		feed, err := s.db.GetFeed(item.FeedId)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		if feed != nil {
			item.Link = htmlutil.AbsoluteUrl(item.Link, feed.Link)
		}
	}

	item.Content = sanitizer.Sanitize(item.Link, item.Content)
	return c.JSON(item)
}

func (s *Server) handleItemUpdate(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body struct {
		Status *storage.ItemStatus `json:"status"`
	}
	if err = c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if body.Status != nil {
		err = s.db.UpdateItemStatus(id, *body.Status)
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleSettings(c fiber.Ctx) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	settings["rsshub_path"] = s.base.Path
	return c.JSON(settings)
}

func (s *Server) handleSettingsUpdate(c fiber.Ctx) error {
	var settings map[string]any
	if err := c.Bind().JSON(&settings); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err := s.db.UpdateSettings(settings)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	if _, ok := settings["refresh_rate"]; ok {
		refreshRate, err := s.db.GetSettingsValueInt64("refresh_rate")
		if err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		go s.SetRefreshRate(refreshRate)
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleOPMLImport(c fiber.Ctx) error {
	fh, err := c.FormFile("opml")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	file, err := fh.Open()
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	folder, err := opml.Parse(file)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	var errs []error
	for _, f := range folder.Feeds {
		_, err = s.db.CreateFeed(f.Title, "", f.SiteUrl, f.FeedUrl, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}
	for _, f := range folder.Folders {
		folder, err := s.db.CreateFolder(f.Title)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, f := range f.AllFeeds() {
			_, err = s.db.CreateFeed(f.Title, "", f.SiteUrl, f.FeedUrl, &folder.Id)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}
	if len(errs) > 0 {
		return c.Status(http.StatusInternalServerError).JSON(errors.Join(errs...))
	}

	go s.FindFavicons()
	go s.RefreshAllFeeds()
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleOPMLExport(c fiber.Ctx) error {
	c.Response().Header.SetContentType("application/xml; charset=utf-8")
	c.Response().Header.Set("Content-Disposition", `attachment; filename="subscriptions.opml"`)

	var f opml.Folder

	feedsByFolderId := make(map[int64][]*storage.Feed)
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	for _, feed := range feeds {
		if feed.FolderId == nil {
			f.Feeds = append(f.Feeds, opml.Feed{
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
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	for _, folder := range folders {
		feeds := feedsByFolderId[folder.Id]
		if len(feeds) == 0 {
			continue
		}
		folder := opml.Folder{Title: folder.Title}
		for _, feed := range feeds {
			folder.Feeds = append(folder.Feeds, opml.Feed{
				Title:   feed.Title,
				FeedUrl: feed.FeedLink,
				SiteUrl: feed.Link,
			})
		}
		f.Folders = append(f.Folders, folder)
	}

	_, err = c.WriteString(f.OPML())
	return err
}
