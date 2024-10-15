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
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/singleflight"
)

const (
	routeExpire   = 5 * time.Minute
	contentExpire = time.Hour
)

type RSSHub struct {
	srcUrl, routesUrl string
	cache             *cache.Cache
	client            http.Client
	modules           map[string]*goja.Program
	files             map[string]string
	mu                sync.Mutex
	g                 singleflight.Group
}

func NewRSSHub(cache *cache.Cache, routesUrl, srcUrl string) *RSSHub {
	return &RSSHub{
		srcUrl:    srcUrl,
		routesUrl: routesUrl,
		cache:     cache,
		client:    http.Client{Timeout: 30 * time.Second},
		modules:   make(map[string]*goja.Program),
		files:     make(map[string]string),
	}
}

func (r *RSSHub) ClearCachedModules() {
	r.mu.Lock()
	clear(r.modules)
	clear(r.files)
	r.mu.Unlock()
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

func (r *RSSHub) route(path string) (*goja.Program, error) {
	prg, err, _ := r.g.Do(path, func() (interface{}, error) {
		r.mu.Lock()
		if module, ok := r.modules[path]; ok {
			r.mu.Unlock()
			return module, nil
		}
		r.mu.Unlock()

		url, err := url.JoinPath(r.srcUrl, path)
		if err != nil {
			return nil, err
		}
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
		prg, err := goja.Compile(path, utils.BytesToString(result.Code), false)
		if err != nil {
			return nil, err
		}

		r.mu.Lock()
		r.modules[path] = prg
		r.mu.Unlock()
		return prg, nil
	})
	if err != nil {
		return nil, err
	}
	return prg.(*goja.Program), nil
}

func (r *RSSHub) file(path string) (string, error) {
	file, err, _ := r.g.Do(path, func() (any, error) {
		r.mu.Lock()
		if file, ok := r.files[path]; ok {
			r.mu.Unlock()
			return file, nil
		}
		r.mu.Unlock()

		url, err := url.JoinPath(r.srcUrl, path)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		_, body, err := r.do(req, nil)
		if err != nil {
			return nil, err
		}
		file := utils.BytesToString(body)

		r.mu.Lock()
		r.files[path] = file
		r.mu.Unlock()
		return file, nil
	})
	if err != nil {
		return "", err
	}
	return file.(string), nil
}
