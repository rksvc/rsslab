package rsshub

import (
	"crypto/md5"
	"embed"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"rsslab/cache"
	"rsslab/utils"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-resty/resty/v2"
)

//go:embed utils
var lib embed.FS

//go:embed third_party
var third_party embed.FS

const routeCacheTTL = 6 * time.Hour

const nodeModulesPrefix = "node_modules/"
const rootPrefix = "@/"

func init() {
	require.RegisterCoreModule("assert", func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("strict", func(value goja.Value, message goja.Value) error {
			if value.ToBoolean() {
				return nil
			}
			return errors.New(message.String())
		})
	})
	require.RegisterCoreModule("path", func(vm *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").ToObject(vm)
		o.Set("join", func(elem ...string) string { return path.Join(elem...) })
		o.Set("dirname", func(p string) string { return path.Dir(p) })
	})
	require.RegisterNativeModule("dotenv/config", func(_ *goja.Runtime, _ *goja.Object) {})
	require.RegisterNativeModule("ofetch", func(_ *goja.Runtime, _ *goja.Object) {})
	require.RegisterNativeModule("@/utils/md5", func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func(data string) string { return fmt.Sprintf("%x", md5.Sum([]byte(data))) })
	})
	require.RegisterNativeModule("@/utils/rand-user-agent", func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func() string { return utils.UserAgent })
	})
	require.RegisterNativeModule("@/types", func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("ViewType", vm.NewObject())
	})
	require.RegisterNativeModule("@/utils/logger", func(vm *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").ToObject(vm)
		for _, name := range []string{"debug", "info", "warn", "error", "http"} {
			o.Set(name, func() {})
		}
	})
	require.RegisterNativeModule("@/utils/cache", func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("tryGet", func(args ...goja.Value) (goja.Value, error) {
			tryGet, _ := goja.AssertFunction(vm.Get("$tryGet"))
			return tryGet(goja.Undefined(), args...)
		})
	})
	for _, words := range [][]string{
		{"config", "not", "found"},
		{"invalid", "parameter"},
		{"not", "found"},
		{"reject"},
		{"request", "in", "progress"},
	} {
		var name string
		for _, word := range words {
			name += strings.ToUpper(word[:1]) + word[1:]
		}
		name += "Error"
		path := rootPrefix + "errors/types/" + strings.Join(words, "-")
		prg, err := goja.Compile(path, fmt.Sprintf("class %s extends Error{name='%s'}", name, name), true)
		if err != nil {
			log.Fatal(err)
		}
		require.RegisterNativeModule(path, func(vm *goja.Runtime, module *goja.Object) {
			result, err := vm.RunProgram(prg)
			if err != nil {
				panic(err)
			}
			module.Set("exports", result)
		})
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

type RSSHub struct {
	srcUrl, routesUrl string
	contentCacheTTL   time.Duration
	cache             *cache.Cache
	client            *resty.Client
	registry          atomic.Value
}

func NewRSSHub(cache *cache.Cache, routesUrl, srcUrl string) *RSSHub {
	r := &RSSHub{
		srcUrl:          srcUrl,
		routesUrl:       routesUrl,
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
				log.Printf("%s, retry attempt %d", err, r.Request.Attempt)
			}),
	}
	r.ResetRegistry()
	return r
}

func (r *RSSHub) ResetRegistry() {
	registry := require.NewRegistryWithLoader(r.sourceLoader)
	registry.RegisterNativeModule("@/utils/render", func(vm *goja.Runtime, module *goja.Object) {
		art := require.Require(vm, "art-template").ToObject(vm)
		render := vm.ToValue(func(filename string, content goja.Value) (goja.Value, error) {
			source, err := r.file(filename)
			if err != nil {
				return goja.Undefined(), err
			}
			render, _ := goja.AssertFunction(art.Get("render"))
			return render(goja.Undefined(), vm.ToValue(string(source)), content, vm.ToValue(map[string]bool{
				"debug":    false,
				"minimize": false,
			}))
		}).ToObject(vm)
		render.Set("defaults", art.Get("defaults"))
		module.Get("exports").ToObject(vm).Set("art", render)
	})
	r.registry.Store(registry)
}

func (r *RSSHub) sourceLoader(p string) ([]byte, error) {
	name := strings.ReplaceAll(p, nodeModulesPrefix, "")

	if i := strings.LastIndex(name, rootPrefix); i != -1 {
		name := name[i+len(rootPrefix):]
		if name == "config" {
			return r.route("lib/config.ts")
		}
		data, err := lib.ReadFile(name + ".js")
		if err != nil {
			return nil, fmt.Errorf("require %s: %s", rootPrefix+name, require.ModuleFileDoesNotExistError)
		}
		return data, nil
	}

	if i := strings.LastIndex(p, nodeModulesPrefix); i != -1 {
		data, err := third_party.ReadFile(path.Join("third_party", p[i+len(nodeModulesPrefix):]+".js"))
		if err == nil {
			return data, nil
		}
	}

	return r.route(path.Join("lib/routes", name+".ts"))
}

var dynamicImport = regexp.MustCompile(`await import\(.+?\)`)

func (r *RSSHub) route(path string) ([]byte, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.cache.TryGet(url, routeCacheTTL, false, func() (any, error) {
		resp, err := r.client.NewRequest().Get(url)
		if err != nil {
			return nil, err
		}

		code := strings.ReplaceAll(resp.String(), "import.meta.url", `"`+path+`"`)
		if path == "lib/config.ts" {
			code = dynamicImport.ReplaceAllLiteralString(code, "{}")
		}
		result := api.Transform(code, api.TransformOptions{
			Sourcefile:     path,
			Format:         api.FormatCommonJS,
			Loader:         api.LoaderTS,
			Sourcemap:      api.SourceMapInline,
			SourcesContent: api.SourcesContentExclude,
			Target:         api.ES2017,
		})
		if len(result.Errors) > 0 {
			return nil, utils.Errorf(result.Errors)
		} else if len(result.Warnings) > 0 {
			log.Print(utils.Errorf(result.Warnings))
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
	data, err := r.cache.TryGet(url, routeCacheTTL, false, func() (any, error) {
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
