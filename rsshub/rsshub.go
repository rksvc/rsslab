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
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-resty/resty/v2"
)

type routes map[string]struct {
	Routes map[string]struct {
		Location string `json:"location"`
	} `json:"routes"`
}

type RSSHub struct {
	srcUrl, routesUrl string
	routeCacheTTL     time.Duration
	contentCacheTTL   time.Duration
	cache             *cache.Cache
	client            *resty.Client
	routes            routes
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

func NewRSSHub(cache *cache.Cache, routesUrl, srcUrl string) *RSSHub {
	return &RSSHub{
		srcUrl:          srcUrl,
		routesUrl:       routesUrl,
		routeCacheTTL:   6 * time.Hour,
		contentCacheTTL: time.Hour,
		cache:           cache,
		client: resty.
			New().
			SetHeader("User-Agent", utils.UserAgent).
			SetTimeout(time.Minute).
			SetRetryCount(2).
			OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
				if r.IsError() {
					return utils.ResponseError(r)
				}
				return nil
			}).
			AddRetryCondition(func(r *resty.Response, err error) bool {
				if r != nil {
					_, ok := retryStatusCodes[r.StatusCode()]
					return ok
				}
				return err != nil
			}).
			AddRetryHook(func(r *resty.Response, err error) {
				log.Printf(`%s, retry attempt %d`, err, r.Request.Attempt)
			}),
	}
}

func (r *RSSHub) LoadRoutes() error {
	var routes routes
	v, err := r.cache.TryGet(r.routesUrl, r.routeCacheTTL, false, func() (any, error) {
		resp, err := r.client.NewRequest().Get(r.routesUrl)
		if err != nil {
			return nil, err
		}
		return resp.Body(), nil
	})
	if err != nil {
		return err
	}
	err = json.Unmarshal(v.([]byte), &routes)
	if err != nil {
		return err
	}
	r.routes = routes
	return nil
}

var dynamicImport = regexp.MustCompile(`await import\(.+?\)`)

func (r *RSSHub) route(path string) ([]byte, error) {
	rawUrl, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.cache.TryGet(rawUrl, r.routeCacheTTL, false, func() (any, error) {
		resp, err := r.client.NewRequest().Get(rawUrl)
		if err != nil {
			return nil, err
		}

		code := strings.ReplaceAll(resp.String(), "import.meta.url", `"`+path+`"`)
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
			return nil, errorf(result.Errors)
		} else if len(result.Warnings) > 0 {
			log.Print(errorf(result.Warnings))
		}
		return result.Code, nil
	})
	if err != nil {
		return nil, err
	}
	return data.([]byte), nil
}

func (r *RSSHub) file(path string) ([]byte, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.cache.TryGet(url, r.routeCacheTTL, false, func() (any, error) {
		resp, err := r.client.NewRequest().Get(url)
		if err != nil {
			return nil, err
		}
		return resp.Body(), nil
	})
	if err != nil {
		return nil, err
	}
	return data.([]byte), nil
}

func errorf(messages []api.Message) error {
	errs := make([]error, 0, len(messages))
	for _, m := range messages {
		var err error
		if m.Location == nil {
			err = errors.New(m.Text)
		} else {
			err = fmt.Errorf("%s:%d:%d %s, %s",
				m.Location.File, m.Location.Line,
				m.Location.Column, m.Text, m.Location.LineText)
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
