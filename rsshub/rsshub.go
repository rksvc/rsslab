package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"rsslab/cache"
	"rsslab/utils"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-resty/resty/v2"
)

type RSSHub struct {
	*resty.Client

	routesUrl, srcUrl        string
	routeCache, contentCache *cache.Cache
	routes                   map[string]struct {
		Routes map[string]struct {
			Location string `json:"location"`
		} `json:"routes"`
	}
}

var retryStatusCodes = map[int]struct{}{
	http.StatusRequestTimeout:      {},
	http.StatusConflict:            {},
	http.StatusTooEarly:            {},
	http.StatusTooManyRequests:     {},
	http.StatusInternalServerError: {},
	http.StatusBadGateway:          {},
	http.StatusServiceUnavailable:  {},
	http.StatusGatewayTimeout:      {},
}

func NewRSSHub(c cache.ICache, routesUrl, srcUrl string) (*RSSHub, error) {
	r := &RSSHub{
		routesUrl:    routesUrl,
		srcUrl:       srcUrl,
		routeCache:   cache.NewCache(c, 6*time.Hour),
		contentCache: cache.NewCache(c, time.Hour),

		Client: resty.
			New().
			SetHeader("User-Agent", utils.UserAgent).
			SetHeader("Accept-Language", utils.AcceptLanguage).
			SetTimeout(30 * time.Second).
			SetRetryCount(2).
			AddRetryCondition(func(r *resty.Response, err error) bool {
				if err != nil {
					return true
				}
				_, ok := retryStatusCodes[r.StatusCode()]
				return ok
			}).
			AddRetryHook(func(r *resty.Response, err error) {
				if err == nil {
					log.Printf(`%s "%s": %s, retry attempt %d`, r.Request.Method, r.Request.URL, r.Status(), r.Request.Attempt)
				}
			}),
	}

	v, err := r.routeCache.TryGet(r.routesUrl, false, func() (any, error) {
		resp, err := r.R().Get(r.routesUrl)
		if err != nil {
			return nil, err
		} else if status := resp.StatusCode(); status < 200 || status >= 300 {
			return nil, fmt.Errorf(`%s "%s": %s`, resp.Request.Method, r.routesUrl, resp.Status())
		}
		return resp.Body(), nil
	})
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(v.([]byte), &r.routes)
	if err != nil {
		return nil, err
	}
	return r, nil
}

var dynamicImport = regexp.MustCompile(`await import\(.+?\)`)

func (r *RSSHub) route(path string) ([]byte, error) {
	rawUrl, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.routeCache.TryGet(rawUrl, false, func() (any, error) {
		resp, err := r.R().Get(rawUrl)
		if err != nil {
			return nil, err
		} else if status := resp.StatusCode(); status < 200 || status >= 300 {
			return nil, fmt.Errorf(`%s "%s": %s`, resp.Request.Method, rawUrl, resp.Status())
		}

		code := resp.String()
		if path == "lib/config.ts" {
			code = dynamicImport.ReplaceAllLiteralString(code, "{}")
		}
		url, err := url.Parse(rawUrl)
		if err != nil {
			return nil, err
		}
		result := api.Transform(code, api.TransformOptions{
			Sourcefile:     url.Path,
			Format:         api.FormatCommonJS,
			Loader:         api.LoaderTS,
			Sourcemap:      api.SourceMapInline,
			SourcesContent: api.SourcesContentExclude,
			Target:         api.ES2017,
		})
		if len(result.Errors) > 0 {
			return nil, errorf(result.Errors...)
		} else if len(result.Warnings) > 0 {
			log.Print(errorf(result.Warnings...))
		}
		return result.Code, nil
	})
	if err != nil {
		return nil, err
	}
	return data.([]byte), nil
}

func errorf(messages ...api.Message) error {
	var errs []error
	for _, m := range messages {
		var err error
		if m.Location == nil {
			err = fmt.Errorf("%s", m.Text)
		} else {
			err = fmt.Errorf("%s:%d:%d %s %s",
				m.Location.File, m.Location.Line,
				m.Location.Column, m.Text, m.Location.LineText)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
