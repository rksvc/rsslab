package rsshub

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cache"
)

func (r *RSSHub) Register(rsshub fiber.Router) {
	rsshub.Use(cache.New(cache.Config{
		Expiration: 5 * time.Minute,
		CacheInvalidator: func(c fiber.Ctx) bool {
			status := c.Response().StatusCode()
			return status < 200 || status >= 300
		},
	}))
	var prefix string
	if g, ok := rsshub.(*fiber.Group); ok {
		prefix = g.Prefix
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
	}

	var total, cnt int
	for name, routes := range r.routes {
		namespace := rsshub.Group(name)
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
					path := strings.TrimPrefix(string(c.Request().URI().Path()), prefix)
					location := strings.TrimSuffix(route.Location, ".ts")

					data := r.Data(name, location, NewCtx(path, params, c.Queries()))
					if err, ok := data.(error); ok {
						status := http.StatusInternalServerError
						if err, ok := err.(*ErrorResponse); ok {
							status = err.Status
						}
						log.Print(err)
						return c.Status(status).SendString(err.Error())
					} else if feed, err := toJSONFeed(data); err != nil {
						log.Print(err)
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					} else {
						return c.JSON(feed, "application/feed+json; charset=UTF-8")
					}
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
