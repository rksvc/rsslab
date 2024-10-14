package rsshub

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	pathpkg "path"
	"rsslab/utils"
	"strings"

	"github.com/dop251/goja"
	"github.com/gofiber/fiber/v2"
)

func (r *RSSHub) Register(app *fiber.App) error {
	v, err := r.cache.TryGet(r.routesUrl, srcExpire, false, func() (any, error) {
		req, err := http.NewRequest(http.MethodGet, r.routesUrl, nil)
		if err != nil {
			return nil, err
		}
		_, body, err := r.do(req, nil)
		return body, err
	})
	if err != nil {
		return err
	}
	var routes map[string]struct {
		Routes map[string]struct {
			Location string `json:"location"`
		} `json:"routes"`
	}
	err = json.Unmarshal(v.([]byte), &routes)
	if err != nil {
		return err
	}

	var total, cnt int
	for namespace, routes := range routes {
		group := app.Group(namespace)
		total += len(routes.Routes)
	register:
		for path, route := range routes.Routes {
			register := func(path, extraParam, key string) {
				group.Get(path, func(c *fiber.Ctx) error {
					params := make(map[string]string)
					var err error
					for _, param := range c.Route().Params {
						if value := c.Params(param); value != "" {
							params[param], err = url.PathUnescape(value)
							if err != nil {
								return c.Status(http.StatusBadRequest).SendString(err.Error())
							}
						}
					}
					if extraParam != "" {
						if value := c.Params(key); value != "" {
							params[extraParam], err = url.PathUnescape(value)
							if err != nil {
								return c.Status(http.StatusBadRequest).SendString(err.Error())
							}
						}
					}
					path := pathpkg.Join(namespace, strings.TrimSuffix(route.Location, ".ts"))
					data, err := r.handle(path, &ctx{Req: req{
						Path:    utils.BytesToString(c.Request().URI().Path()),
						queries: c.Queries(),
						params:  params,
					}})
					if err != nil {
						if e, ok := err.(*goja.Exception); ok {
							err = errors.New(e.String())
						}
						log.Print(err)
						return c.Status(http.StatusInternalServerError).SendString(err.Error())
					}
					if c.QueryBool("debug") {
						return c.JSON(data)
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
					{"{.+}", "+"},
					{"{.*}", "*"},
					{"{.+}?", "*"},
					{"{.*}?", "*"},
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
				log.Printf("skipped %s%s", namespace, path)
			} else {
				register(path, "", "")
			}
		}
	}
	log.Printf("registered %d routes, skipped %d", cnt, total-cnt)
	return nil
}
