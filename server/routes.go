package server

import (
	"encoding/xml"
	"errors"
	"log"
	"net/http"
	"net/url"
	"rsslab/storage"
	"rsslab/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/net/html/charset"
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
	api.Get("/transform/:type/:params", s.handleTransform)
}

func (s *Server) handleStatus(c *fiber.Ctx) error {
	stats, err := s.db.FeedStats()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(fiber.Map{
		"stats":          stats,
		"running":        s.pending.Load(),
		"last_refreshed": s.lastRefreshed.Load(),
	})
}

func (s *Server) handleFolderList(c *fiber.Ctx) error {
	folders, err := s.db.ListFolders()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(folders)
}

func (s *Server) handleFolderCreate(c *fiber.Ctx) error {
	var body struct {
		Title string `json:"title"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	folder, err := s.db.CreateFolder(body.Title)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.Status(http.StatusCreated).JSON(folder)
}

func (s *Server) handleFolderUpdate(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var editor storage.FolderEditor
	if err = c.BodyParser(&editor); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err = s.db.EditFolder(id, editor); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderDelete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if err = s.db.DeleteFolder(id); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFolderRefresh(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
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

func (s *Server) handleFeedList(c *fiber.Ctx) error {
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(feeds)
}

func (s *Server) handleFeedCreate(c *fiber.Ctx) error {
	var body struct {
		Url      string `json:"url"`
		FolderId *int   `json:"folder_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	var state storage.HTTPState
	rawFeed, err := s.do(body.Url, &state)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	rawFeed.FeedLink = body.Url
	feed, err := s.db.CreateFeed(
		rawFeed.Title,
		rawFeed.Link,
		rawFeed.FeedLink,
		body.FolderId,
	)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	go s.FindFeedFavicon(*feed)

	items := convertItems(rawFeed.Items, *feed)
	lastRefreshed := time.Now()
	if err = s.db.CreateItems(items, feed.Id, lastRefreshed, &state); err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}

	feed.LastRefreshed = &lastRefreshed
	return c.JSON(feed)
}

func (s *Server) handleFeedsRefresh(c *fiber.Ctx) error {
	go s.RefreshAllFeeds()
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedErrorsList(c *fiber.Ctx) error {
	errors, err := s.db.GetFeedErrors()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(errors)
}

func (s *Server) handleFeedIcon(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.SendStatus(http.StatusBadRequest)
	}

	bytes, err := s.db.GetFeedIcon(id)
	if err != nil {
		log.Print(err)
		return c.SendStatus(http.StatusInternalServerError)
	} else if bytes == nil {
		return c.SendStatus(http.StatusNotFound)
	}

	c.Set("Content-Type", http.DetectContentType(bytes))
	c.Set("Cache-Control", "max-age=86400") // one day
	return c.Send(bytes)
}

func (s *Server) handleFeedRefresh(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	feed, err := s.db.GetFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	} else if feed == nil {
		return c.Status(http.StatusNotFound).SendString("no such feed")
	}
	go s.RefreshFeeds(*feed)
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedUpdate(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body struct {
		Title    *string `json:"title"`
		FeedLink *string `json:"feed_link"`
		FolderId *int    `json:"folder_id"`
	}
	if err = c.BodyParser(&body); err != nil {
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
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleFeedDelete(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err = s.db.DeleteFeed(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItemList(c *fiber.Ctx) error {
	var filter storage.ItemFilter
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	oldestFirst := c.QueryBool("oldest_first")

	const PER_PAGE = 20
	items, err := s.db.ListItems(filter, PER_PAGE+1, oldestFirst)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
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

func (s *Server) handleItemRead(c *fiber.Ctx) error {
	var filter storage.ItemFilter
	if err := c.QueryParser(&filter); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	err := s.db.MarkItemsRead(filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleItem(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	item, err := s.db.GetItem(id)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	} else if item == nil {
		return c.Status(http.StatusNotFound).SendString("no such item")
	}
	item.Content = utils.Sanitize(item.Link, item.Content)
	return c.JSON(item)
}

func (s *Server) handleItemUpdate(c *fiber.Ctx) error {
	id, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var body struct {
		Status *storage.ItemStatus `json:"status"`
	}
	if err = c.BodyParser(&body); err != nil {
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

func (s *Server) handleSettings(c *fiber.Ctx) error {
	settings, err := s.db.GetSettings()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(settings)
}

func (s *Server) handleSettingsUpdate(c *fiber.Ctx) error {
	var editor storage.SettingsEditor
	if err := c.BodyParser(&editor); err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	if editor.RefreshRate != nil {
		if err := s.db.UpdateSettings(editor); err != nil {
			return c.Status(http.StatusInternalServerError).SendString(err.Error())
		}
		go s.SetRefreshRate(*editor.RefreshRate)
	}
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleOPMLImport(c *fiber.Ctx) error {
	fh, err := c.FormFile("opml")
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	file, err := fh.Open()
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	d := xml.NewDecoder(file)
	d.Entity = xml.HTMLEntity
	d.Strict = false
	d.CharsetReader = charset.NewReaderLabel
	var opml OPML
	err = d.Decode(&opml)
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}

	var errs []error
	for _, o := range opml.Outlines {
		if o.isFolder() {
			title := o.Title
			if title == "" {
				title = o.Title2
			}
			folder, err := s.db.CreateFolder(title)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			for _, o := range o.allFeeds() {
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
		return c.Status(http.StatusInternalServerError).JSON(errors.Join(errs...))
	}

	go s.FindFavicons()
	go s.RefreshAllFeeds()
	return c.SendStatus(http.StatusOK)
}

func (s *Server) handleOPMLExport(c *fiber.Ctx) error {
	opml := OPML{
		Version: "1.1",
		Title:   "subscriptions",
	}

	feedsByFolderId := make(map[int][]*storage.Feed)
	feeds, err := s.db.ListFeeds()
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	for _, feed := range feeds {
		if feed.FolderId == nil {
			opml.Outlines = append(opml.Outlines, Outline{
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
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	for _, folder := range folders {
		feeds := feedsByFolderId[folder.Id]
		if len(feeds) == 0 {
			continue
		}
		folder := Outline{Title: folder.Title}
		for _, feed := range feeds {
			folder.Outlines = append(folder.Outlines, Outline{
				Type:    "rss",
				Title:   feed.Title,
				FeedUrl: feed.FeedLink,
				SiteUrl: feed.Link,
			})
		}
		opml.Outlines = append(opml.Outlines, folder)
	}

	c.Set("Content-Type", "application/xml; charset=utf-8")
	c.Set("Content-Disposition", `attachment; filename="subscriptions.opml"`)
	_, err = c.WriteString(xml.Header)
	if err != nil {
		return err
	}
	e := xml.NewEncoder(c.Response().BodyWriter())
	e.Indent("", "  ")
	return e.Encode(opml)
}

func (s *Server) handleTransform(c *fiber.Ctx) error {
	params, err := url.PathUnescape(c.Params("params"))
	if err != nil {
		return c.Status(http.StatusBadRequest).SendString(err.Error())
	}
	var state storage.HTTPState
	feed, err := s.do(c.Params("type")+":"+params, &state)
	if err != nil {
		return c.Status(http.StatusInternalServerError).SendString(err.Error())
	}
	return c.JSON(feed, "application/feed+json; charset=UTF-8")
}
