package rsshub

import (
	"log"
	"net/http"
	"rsslab/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
)

func (r *RSSHub) Register(app *fiber.App) {
	app.Use(cache.New(cache.Config{
		Expiration: 5 * time.Minute,
		CacheInvalidator: func(c fiber.Ctx) bool {
			return utils.IsErrorResponse(c.Response().StatusCode())
		},
	}))

	var total, cnt int
	for name, routes := range r.routes {
		namespace := app.Group(name)
		total += len(routes.Routes)
	register:
		for path, route := range routes.Routes {
			register := func(path, extraParam, key string) {
				namespace.Get(path, func(c fiber.Ctx) error {
					params := make(map[string]string)
					for _, param := range c.Route().Params {
						if value := c.Params(param); value != "" {
							params[param] = value
						}
					}
					if extraParam != "" {
						if value := c.Params(key); value != "" {
							params[extraParam] = value
						}
					}
					path := string(c.Request().URI().Path())
					location := strings.TrimSuffix(route.Location, ".ts")

					data, err := r.Data(name, location, NewCtx(path, params, c.Queries()))
					if err != nil {
						log.Print(err)
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					}
					feed, err := toJSONFeed(data)
					if err != nil {
						log.Print(err)
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					}
					return c.JSON(feed, "application/feed+json; charset=UTF-8")
				})
				cnt++
			}

			if strings.ContainsRune(path, '{') {
				for _, pk := range []struct{ pattern, key string }{
					{"{.+}?", "*"},
					{"{.*}?", "*"},
					{"{.+}", "+"},
				} {
					if before, found := strings.CutSuffix(path, pk.pattern); found {
						if strings.ContainsRune(before, '{') {
							break
						}
						i := strings.LastIndexByte(before, '/')
						var extraParam string
						if after, found := strings.CutPrefix(before[i+1:], ":"); found {
							extraParam = after
						}
						register(before[:i+1]+pk.key, extraParam, pk.key)
						continue register
					}
				}
				log.Printf("skipped %s%s", name, path)
			} else {
				register(path, "", "")
			}
		}
	}
	log.Printf("registered %d routes, skipped %d", cnt, total-cnt)
}
