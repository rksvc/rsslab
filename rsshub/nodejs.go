package rsshub

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"path"
	"rsslab/utils"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/url"
)

type moduleLoader func(*goja.Object, *requireModule)

type requireModule struct {
	r       *RSSHub
	vm      *goja.Runtime
	jobs    chan<- func()
	modules map[string]goja.Value
}

//go:embed utils
var lib embed.FS

//go:embed third_party
var third_party embed.FS

var core = make(map[string]*goja.Program)
var native = map[string]moduleLoader{
	// Node.js modules
	"assert": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("strict", func(value goja.Value, message goja.Value) error {
			if value.ToBoolean() {
				return nil
			}
			return errors.New(message.String())
		})
	},
	"path": func(module *goja.Object, r *requireModule) {
		o := module.Get("exports").ToObject(r.vm)
		o.Set("join", func(elem ...string) string { return path.Join(elem...) })
		o.Set("dirname", func(p string) string { return path.Dir(p) })
	},
	"url": func(module *goja.Object, r *requireModule) {
		url.Require(r.vm, module)
		module.Get("exports").ToObject(r.vm).Set("fileURLToPath", func(url string) string { return url })
	},

	// RSSHub dependencies
	"dotenv/config": func(_ *goja.Object, _ *requireModule) {},
	"ofetch":        func(_ *goja.Object, _ *requireModule) {},

	// RSSHub source files
	"@/types": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("ViewType", r.vm.NewObject())
	},
	"@/utils/md5": func(module *goja.Object, _ *requireModule) {
		module.Set("exports", func(data string) string { return fmt.Sprintf("%x", md5.Sum(utils.StringToBytes(data))) })
	},
	"@/utils/rand-user-agent": func(module *goja.Object, _ *requireModule) {
		module.Set("exports", func() string { return utils.USER_AGENT })
	},
	"@/utils/logger": func(module *goja.Object, r *requireModule) {
		o := module.Get("exports").ToObject(r.vm)
		for _, name := range []string{"debug", "info", "warn", "error", "http"} {
			o.Set(name, func() {})
		}
	},
	"@/utils/ofetch": func(module *goja.Object, r *requireModule) {
		ofetch := r.vm.ToValue(func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtOfetch, r) }).ToObject(r.vm)
		ofetch.Set("raw", func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtOfetchRaw, r) })
		module.Set("exports", ofetch)
	},
	"@/utils/got": func(module *goja.Object, r *requireModule) {
		got := r.vm.ToValue(func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtGot, r) }).ToObject(r.vm)
		for _, method := range []string{"get", "post", "put", "head", "patch", "delete"} {
			got.Set(method, func(req, opts goja.Value) *goja.Promise {
				return fetch(req, opts, strings.ToUpper(method), respFmtGot, r)
			})
		}
		module.Set("exports", got)
	},
	"@/utils/cache": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("tryGet", func(key string, f func() goja.Value, maxAge *int, ex *bool) *goja.Promise {
			promise, resolve, reject := r.vm.NewPromise()
			go func() {
				ttl := contentExpire
				if maxAge != nil {
					ttl = time.Duration(*maxAge) * time.Second
				}
				v, err := r.r.cache.TryGet(key, ttl, ex == nil || *ex, func() (any, error) {
					var w wait
					w.Add(1)
					r.jobs <- func() { w.Await(r.vm, f()) }
					w.Wait()
					return w.Value, w.Err
				})
				var data any
				if b, ok := v.([]byte); !ok {
					data = v
				} else if json.Unmarshal(b, &data) != nil {
					data = utils.BytesToString(b)
				}
				r.jobs <- func() {
					if err == nil {
						resolve(data)
					} else {
						reject(err)
					}
				}
			}()
			return promise
		})
	},
}

var errNoSuchModule = errors.New("no such module")

func init() {
	for _, words := range [][]string{
		{"config", "not", "found"},
		{"invalid", "parameter"},
		{"not", "found"},
		{"reject"},
		{"request", "in", "progress"},
	} {
		path := "@/errors/types/" + strings.Join(words, "-")
		native[path] = func(module *goja.Object, r *requireModule) {
			var name string
			for _, word := range words {
				name += strings.ToUpper(word[:1]) + word[1:]
			}
			name += "Error"
			val, err := r.vm.RunScript(path, fmt.Sprintf("(class extends Error{name='%s'})", name))
			if err != nil {
				log.Fatal(err)
			}
			module.Set("exports", val)
		}
	}

	fs.WalkDir(lib, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil
		}
		src, err := fs.ReadFile(lib, path)
		if err != nil {
			log.Fatal(err)
		}
		name := "@/" + strings.TrimSuffix(path, ".js")
		prg, err := goja.Compile(name, utils.BytesToString(src), false)
		if err != nil {
			log.Fatal(err)
		}
		core[name] = prg
		return nil
	})

	fs.WalkDir(third_party, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		if d.IsDir() {
			return nil
		}
		src, err := fs.ReadFile(third_party, path)
		if err != nil {
			log.Fatal(err)
		}
		name := strings.TrimSuffix(d.Name(), ".js")
		prg, err := goja.Compile(name, utils.BytesToString(src), false)
		if err != nil {
			log.Fatal(err)
		}
		core[name] = prg
		return nil
	})
}

func (r *requireModule) require(p string) (goja.Value, error) {
	p = r.resolve(p)
	if module, ok := r.modules[p]; ok {
		return module, nil
	}

	var module *goja.Object
	var err error

	if strings.HasPrefix(p, "/") {
		r.r.mu.Lock()
		prg, ok := r.r.modules[p]
		if !ok {
			r.r.mu.Unlock()
			src, err := r.r.route(p)
			if err != nil {
				return nil, err
			}
			prg, err = goja.Compile(p, src, false)
			if err != nil {
				return nil, err
			}
			r.r.mu.Lock()
			r.r.modules[p] = prg
		}
		r.r.mu.Unlock()
		module, err = loadIIFEModule(prg, r.vm)
		if err != nil {
			return nil, err
		}

		if p == "/lib/config.ts" {
			exports := module.Get("exports").ToObject(r.vm)
			config := exports.Get("config").ToObject(r.vm)

			cache := config.Get("cache").ToObject(r.vm)
			cache.Set("routeExpire", routeExpire/time.Second)
			cache.Set("contentExpire", contentExpire/time.Second)

			config.Get("feature").ToObject(r.vm).Set("allow_user_supply_unsafe_domain", true)
			config.Set("ua", utils.USER_AGENT)
		}

	} else if prg, ok := core[p]; ok {
		module, err = loadIIFEModule(prg, r.vm)
		if err != nil {
			return nil, err
		}

	} else if ldr, ok := native[p]; ok {
		module = r.vm.NewObject()
		module.Set("exports", r.vm.NewObject())
		ldr(module, r)

	} else if p == "@/utils/render" {
		// not in `native` due to initialization cycle
		v, err := r.require("art-template")
		if err != nil {
			return nil, err
		}
		art := v.ToObject(r.vm)
		render := r.vm.ToValue(func(filename string, data goja.Value) (goja.Value, error) {
			source, err := r.r.file(filename)
			if err != nil {
				return nil, err
			}
			render, _ := goja.AssertFunction(art.Get("render"))
			return render(goja.Undefined(), r.vm.ToValue(source), data, r.vm.ToValue(map[string]any{
				"debug":    false,
				"minimize": false,
			}))
		}).ToObject(r.vm)
		render.Set("defaults", art.Get("defaults"))

		exports := r.vm.NewObject()
		exports.Set("art", render)
		module = r.vm.NewObject()
		module.Set("exports", exports)
	}

	if module != nil {
		module := module.Get("exports")
		r.modules[p] = module
		return module, nil
	}
	return nil, fmt.Errorf("require %s: %w", p, errNoSuchModule)
}

func (r *requireModule) resolve(p string) string {
	if strings.HasPrefix(p, ".") {
		var buf [2]goja.StackFrame
		frames := r.vm.CaptureCallStack(2, buf[:0])
		dir := "/lib/routes"
		if len(frames) > 1 {
			if srcName := frames[1].SrcName(); srcName != "" {
				dir = path.Dir(srcName)
			}
		}
		return path.Join(dir, p+".ts")
	} else if after, found := strings.CutPrefix(p, "node:"); found {
		return after
	} else if p == "@/config" {
		return "/lib/config.ts"
	}
	return p
}

func loadIIFEModule(prg *goja.Program, vm *goja.Runtime) (*goja.Object, error) {
	f, err := vm.RunProgram(prg)
	if err != nil {
		return nil, err
	}
	call, _ := goja.AssertFunction(f)
	module := vm.NewObject()
	exports := vm.NewObject()
	module.Set("exports", exports)
	_, err = call(exports, exports, vm.Get("require"), module)
	return module, err
}

func fetch(req, opts goja.Value, method string, respFmt respFmt, r *requireModule) *goja.Promise {
	promise, resolve, reject := r.vm.NewPromise()
	go func() {
		resp, err := r.r.fetch(req, opts, method, respFmt, r.vm)
		r.jobs <- func() {
			if err == nil {
				resolve(resp)
			} else {
				reject(err)
			}
		}
	}()
	return promise
}
