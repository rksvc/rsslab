package server

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"net/http"
	"rsslab/storage"
	"rsslab/utils"
	"strconv"
	"time"

	"github.com/go-errors/errors"
	"github.com/gofiber/fiber/v3"
	"github.com/mmcdole/gofeed"
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
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.JSON(fiber.Map{
		"stats":   stats,
		"running": s.pending.Load(),
	})
}

func (s *Server) handleFolderList(c fiber.Ctx) error {
	folders, err := s.db.ListFolders()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
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
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.Status(http.StatusCreated).JSON(folder)
}

func (s *Server) handleFolderUpdate(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var editor storage.FolderEditor
	if err = c.Bind().JSON(&editor); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err = s.db.EditFolder(id, editor); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderDelete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err = s.db.DeleteFolder(id); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderRefresh(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	feeds, err := s.db.GetFeeds(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	go s.RefreshFeeds(feeds...)
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedList(c fiber.Ctx) error {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.JSON(feeds)
}

func (s *Server) handleFeedCreate(c fiber.Ctx) error {
	var body struct {
		Url      string `json:"url"`
		FolderId *int   `json:"folder_id"`
	}
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	req, err := http.NewRequest(http.MethodGet, body.Url, nil)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	resp, err := s.do(req)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	defer resp.Body.Close()
	var f io.Reader = resp.Body
	if e := utils.GetEncoding(resp); e != nil {
		f = e.NewDecoder().Reader(f)
	}
	rawFeed, err := gofeed.NewParser().Parse(f)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	rawFeed.FeedLink = body.Url
	feed, err := s.db.CreateFeed(
		rawFeed.Title,
		rawFeed.Link,
		rawFeed.FeedLink,
		body.FolderId,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	go s.FindFeedFavicon(*feed)

	items := convertItems(rawFeed.Items, *feed)
	lastRefreshed := time.Now()
	if err = s.db.CreateItems(items, feed.Id, lastRefreshed, getHTTPState(resp)); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
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
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.JSON(errors)
}

type icon struct {
	ctype string
	bytes []byte
	etag  string
}

func (s *Server) handleFeedIcon(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	val, err := s.cache.TryGet(strconv.Itoa(id), time.Hour, true, func() (any, error) {
		bytes, err := s.db.GetFeedIcon(id)
		if err != nil {
			return nil, err
		} else if bytes == nil {
			return nil, nil
		}
		return &icon{
			bytes: bytes,
			ctype: http.DetectContentType(bytes),
			etag:  fmt.Sprintf("%x", md5.Sum(bytes)),
		}, nil
	})
	if err != nil {
		log.Print(err)
		return c.SendStatus(http.StatusInternalServerError)
	} else if val == nil {
		return c.SendStatus(http.StatusNotFound)
	}

	icon := val.(*icon)
	if string(c.Request().Header.Peek("If-None-Match")) == icon.etag {
		return c.SendStatus(http.StatusNotModified)
	}
	c.Response().Header.SetContentType(icon.ctype)
	c.Response().Header.Set("Etag", icon.etag)
	_, err = c.Write(icon.bytes)
	return err
}

func (s *Server) handleFeedRefresh(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	feed, err := s.db.GetFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	} else if feed == nil {
		return c.Status(http.StatusNotFound).SendString("no such feed")
	}
	go s.RefreshFeeds(*feed)
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedUpdate(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body struct {
		Title    *string `json:"title"`
		FeedLink *string `json:"feed_link"`
		FolderId *int    `json:"folder_id"`
	}
	if err = c.Bind().JSON(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
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
	if err = s.db.EditFeed(id, editor); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedDelete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err = s.db.DeleteFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItemList(c fiber.Ctx) error {
	var filter storage.ItemFilter
	if err := c.Bind().Query(&filter); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	newestFirst := c.Query("oldest_first") != "true"

	const PER_PAGE = 20
	items, err := s.db.ListItems(filter, PER_PAGE+1, newestFirst)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	hasMore := false
	if len(items) > PER_PAGE {
		hasMore = true
		items = items[:PER_PAGE]
	}
	return c.JSON(fiber.Map{
		"list":     items,
		"has_more": hasMore,
	})
}

func (s *Server) handleItemRead(c fiber.Ctx) error {
	var filter storage.ItemFilter
	if err := c.Bind().Query(&filter); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err := s.db.MarkItemsRead(filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItem(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	item, err := s.db.GetItem(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	} else if item == nil {
		return c.Status(http.StatusNotFound).SendString("no such item")
	}
	item.Content = sanitizer.Sanitize(item.Link, item.Content)
	return c.JSON(item)
}

func (s *Server) handleItemUpdate(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
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
			return c.Status(http.StatusInternalServerError).SendString(errString(err))
		}
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleSettings(c fiber.Ctx) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
	}
	return c.JSON(settings)
}

func (s *Server) handleSettingsUpdate(c fiber.Ctx) error {
	var editor storage.SettingsEditor
	if err := c.Bind().JSON(&editor); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if editor.RefreshRate != nil {
		if err := s.db.UpdateSettings(editor); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(errString(err))
		}
		go s.SetRefreshRate(*editor.RefreshRate)
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
		_, err = s.db.CreateFeed(f.Title, f.SiteUrl, f.FeedUrl, nil)
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
			_, err = s.db.CreateFeed(f.Title, f.SiteUrl, f.FeedUrl, &folder.Id)
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
	var f opml.Folder

	feedsByFolderId := make(map[int][]*storage.Feed)
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
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
		return c.Status(http.StatusInternalServerError).SendString(errString(err))
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

	c.Response().Header.SetContentType("application/xml; charset=utf-8")
	c.Response().Header.Set("Content-Disposition", `attachment; filename="subscriptions.opml"`)
	_, err = c.WriteString(f.OPML())
	return err
}
