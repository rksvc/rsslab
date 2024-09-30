package rsshub

import (
	"crypto/md5"
	"embed"
	"errors"
	"fmt"
	"io/fs"
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

const srcExpire = 6 * time.Hour
const routeExpire = 5 * time.Minute
const contentExpire = time.Hour

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
		module.Get("exports").ToObject(vm).Set("tryGet", vm.Get("$tryGet"))
	})

	for _, words := range [][]string{
		{"config", "not", "found"},
		{"invalid", "parameter"},
		{"not", "found"},
		{"reject"},
		{"request", "in", "progress"},
	} {
		path := "@/errors/types/" + strings.Join(words, "-")
		require.RegisterNativeModule(path, func(vm *goja.Runtime, module *goja.Object) {
			var name string
			for _, word := range words {
				name += strings.ToUpper(word[:1]) + word[1:]
			}
			name += "Error"
			prg, err := goja.Compile(path, fmt.Sprintf("(class extends Error{name='%s'})", name), false)
			if err != nil {
				log.Fatal(err)
			}
			result, err := vm.RunProgram(prg)
			if err != nil {
				log.Fatal(err)
			}
			module.Set("exports", result)
		})
	}

	fs.WalkDir(lib, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := "@/" + strings.TrimSuffix(path, ".js")
		require.RegisterNativeModule(name, func(vm *goja.Runtime, module *goja.Object) {
			src, err := fs.ReadFile(lib, path)
			if err != nil {
				log.Fatal(err)
			}
			loadModule(src, name, vm, module)
		})
		return nil
	})
	entries, err := third_party.ReadDir("third_party")
	if err != nil {
		log.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".js")
		require.RegisterNativeModule(name, func(vm *goja.Runtime, module *goja.Object) {
			src, err := third_party.ReadFile(path.Join("third_party", entry.Name()))
			if err != nil {
				log.Fatal(err)
			}
			loadModule(src, name, vm, module)
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
	cache             *cache.Cache
	client            *resty.Client
	registry          atomic.Value
}

func NewRSSHub(cache *cache.Cache, routesUrl, srcUrl string) *RSSHub {
	r := &RSSHub{
		srcUrl:    srcUrl,
		routesUrl: routesUrl,
		cache:     cache,
		client: resty.
			New().
			SetHeader("User-Agent", utils.UserAgent).
			SetTimeout(30 * time.Second).
			SetRetryCount(2).
			OnAfterResponse(func(c *resty.Client, r *resty.Response) error {
				if r.IsError() {
					return utils.ResponseError(r.RawResponse)
				}
				return nil
			}).
			AddRetryCondition(func(r *resty.Response, err error) bool {
				if r != nil && r.StatusCode() > 0 {
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
	name := "@/config"
	registry.RegisterNativeModule(name, func(vm *goja.Runtime, module *goja.Object) {
		src, err := r.route("lib/config.ts")
		if err != nil {
			panic(err)
		}
		loadModule(src, name, vm, module)

		config := module.Get("exports").ToObject(vm).Get("config").ToObject(vm)

		cache := config.Get("cache").ToObject(vm)
		cache.Set("routeExpire", routeExpire/time.Second)
		cache.Set("contentExpire", contentExpire/time.Second)

		config.Get("feature").ToObject(vm).Set("allow_user_supply_unsafe_domain", true)
		config.Set("ua", utils.UserAgent)
	})

	r.registry.Store(registry)
}

func (r *RSSHub) sourceLoader(p string) ([]byte, error) {
	name := strings.ReplaceAll(p, "node_modules/", "")
	if i := strings.LastIndex(name, "@/"); i != -1 {
		return nil, fmt.Errorf("require %s: %s", name[i:], require.ModuleFileDoesNotExistError)
	}
	return r.route(path.Join("lib/routes", name+".ts"))
}

var dynamicImport = regexp.MustCompile(`await import\(.+?\)`)

func (r *RSSHub) route(path string) ([]byte, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.cache.TryGet(url, srcExpire, false, func() (any, error) {
		resp, err := r.client.NewRequest().Get(url)
		if err != nil {
			return nil, err
		}

		code := strings.ReplaceAll(resp.String(), "import.meta.url", `"`+path+`"`)
		if path == "lib/config.ts" {
			code = dynamicImport.ReplaceAllLiteralString(code, "{}")
		}
		result := api.Transform(code, api.TransformOptions{
			Sourcefile:        path,
			Format:            api.FormatCommonJS,
			Loader:            api.LoaderTS,
			Sourcemap:         api.SourceMapInline,
			SourcesContent:    api.SourcesContentExclude,
			Target:            api.ES2017,
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
		return nil, err
	}
	return data.([]byte), nil
}

func (r *RSSHub) file(path string) ([]byte, error) {
	url, err := url.JoinPath(r.srcUrl, path)
	if err != nil {
		return nil, err
	}
	data, err := r.cache.TryGet(url, srcExpire, false, func() (any, error) {
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

func loadModule(src []byte, name string, vm *goja.Runtime, module *goja.Object) {
	const PREFIX = "(function(exports,require,module){"
	const SUFFIX = "})"
	var b strings.Builder
	b.Grow(len(PREFIX) + len(SUFFIX) + len(src))
	b.WriteString(PREFIX)
	b.Write(src)
	b.WriteString(SUFFIX)
	prg, err := goja.Compile(name, b.String(), false)
	if err != nil {
		log.Fatal(err)
	}
	f, err := vm.RunProgram(prg)
	if err != nil {
		log.Fatal(err)
	}
	call, _ := goja.AssertFunction(f)
	exports := module.Get("exports")
	_, err = call(exports, exports, vm.Get("require"), module)
	if err != nil {
		log.Fatal(err)
	}
}
