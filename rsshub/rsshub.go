package rsshub

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"rsslab/cache"
	"rsslab/utils"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	srcExpire     = 6 * time.Hour
	routeExpire   = 5 * time.Minute
	contentExpire = time.Hour
)

type RSSHub struct {
	srcUrl, routesUrl string
	cache             *cache.Cache
	client            http.Client
}

func NewRSSHub(cache *cache.Cache, routesUrl, srcUrl string) *RSSHub {
	return &RSSHub{
		srcUrl:    srcUrl,
		routesUrl: routesUrl,
		cache:     cache,
		client:    http.Client{Timeout: 30 * time.Second},
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

func (r *RSSHub) do(req *http.Request, body any) (resp *http.Response, respBody []byte, err error) {
	switch body := body.(type) {
	case nil, string, []byte:
	default:
		body, err = json.Marshal(body)
		if err != nil {
			return
		}
	}
	const maxTry = 3
	for attempt := 1; attempt <= maxTry; attempt++ {
		switch body := body.(type) {
		case nil:
		case string:
			req.Body = io.NopCloser(strings.NewReader(body))
		case []byte:
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err = r.client.Do(req)
		if err == nil {
			if utils.IsErrorResponse(resp.StatusCode) {
				resp.Body.Close()
				err = utils.ResponseError(resp)
				if _, ok := retryStatusCodes[resp.StatusCode]; !ok {
					return
				}
			} else {
				respBody, err = io.ReadAll(resp.Body)
				resp.Body.Close()
				if err == nil {
					return
				}
			}
		}
		if attempt < maxTry {
			log.Printf("%s, retry attempt %d", err, attempt)
		}
	}
	return
}

func (r *RSSHub) route(path string) (string, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return "", err
	}
	src, err := r.cache.TryGet(url, srcExpire, false, func() (any, error) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		_, body, err := r.do(req, nil)
		if err != nil {
			return nil, err
		}

		code := utils.BytesToString(body)
		if path == "/lib/config.ts" {
			code = strings.Replace(code,
				"import('@/utils/logger')",
				"{                      }", 1)
		}
		result := api.Transform(code, api.TransformOptions{
			Sourcefile:        path,
			Format:            api.FormatCommonJS,
			Loader:            api.LoaderTS,
			Sourcemap:         api.SourceMapInline,
			SourcesContent:    api.SourcesContentExclude,
			Target:            api.ES2023,
			Supported:         utils.SupportedSyntaxFeatures,
			Define:            map[string]string{"import.meta.url": `"` + path + `"`},
			Banner:            utils.IIFE_PREFIX,
			Footer:            utils.IIFE_SUFFIX,
			MinifyWhitespace:  true,
			MinifySyntax:      true,
			MinifyIdentifiers: true,
		})
		if len(result.Errors) > 0 {
			return nil, utils.Errorf(result.Errors)
		} else if len(result.Warnings) > 0 {
			log.Print(utils.Errorf(result.Warnings))
		}
		return result.Code, nil
	})
	if err != nil {
		return "", err
	}
	return utils.BytesToString(src.([]byte)), nil
}

func (r *RSSHub) file(path string) (string, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return "", err
	}
	file, err := r.cache.TryGet(url, srcExpire, false, func() (any, error) {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		_, body, err := r.do(req, nil)
		if err != nil {
			return nil, err
		}
		return body, nil
	})
	if err != nil {
		return "", err
	}
	return utils.BytesToString(file.([]byte)), nil
}
