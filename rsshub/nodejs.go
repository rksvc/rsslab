package rsshub

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"reflect"
	"rsslab/storage"
	"rsslab/utils"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
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
		module.Get("exports").ToObject(r.vm).Set("fileURLToPath", func(call goja.FunctionCall) goja.Value {
			return call.Argument(0)
		})
	},
	"buffer": func(module *goja.Object, r *requireModule) {
		buffer.Require(r.vm, module)
		buffer := module.Get("exports").ToObject(r.vm).Get("Buffer").ToObject(r.vm)
		buffer.Set("isBuffer", func(call goja.FunctionCall) goja.Value {
			return r.vm.ToValue(r.vm.InstanceOf(call.Argument(0), buffer))
		})
	},

	// RSSHub dependencies
	"dotenv/config": func(_ *goja.Object, _ *requireModule) {},
	"ofetch":        func(_ *goja.Object, _ *requireModule) {},
	"sanitize-html": func(module *goja.Object, r *requireModule) {
		sanitize := r.vm.ToValue(func(input string, opts *goja.Object) string {
			if opts == nil || opts.Get("allowedTags").ToObject(r.vm).Get("length").ToBoolean() {
				return utils.Sanitize("", input)
			}
			return utils.ExtractText(input)
		})
		sanitize.ToObject(r.vm).Set("defaults", map[string]any{
			"allowedTags": r.vm.NewArray(),
		})
		module.Get("exports").ToObject(r.vm).Set("default", sanitize)
	},
	// specifically for /copymanga/comic
	"tiny-async-pool": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("default", func() goja.Value {
			return r.vm.NewArray()
		})
	},

	// RSSHub source files
	"@/types": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("ViewType", r.vm.NewObject())
	},
	"@/utils/md5": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("default", func(data string) string {
			return fmt.Sprintf("%x", md5.Sum(utils.StringToBytes(data)))
		})
	},
	"@/utils/rand-user-agent": func(module *goja.Object, r *requireModule) {
		module.Get("exports").ToObject(r.vm).Set("default", func() string { return utils.USER_AGENT })
	},
	"@/utils/logger": func(module *goja.Object, r *requireModule) {
		o := r.vm.NewObject()
		for _, name := range []string{"debug", "info", "warn", "error", "http"} {
			o.Set(name, func() {})
		}
		module.Get("exports").ToObject(r.vm).Set("default", o)
	},
	"@/utils/ofetch": func(module *goja.Object, r *requireModule) {
		ofetch := r.vm.ToValue(func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtOfetch, r) }).ToObject(r.vm)
		ofetch.Set("raw", func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtOfetchRaw, r) })
		module.Get("exports").ToObject(r.vm).Set("default", ofetch)
	},
	"@/utils/got": func(module *goja.Object, r *requireModule) {
		got := r.vm.ToValue(func(req, opts goja.Value) *goja.Promise { return fetch(req, opts, "", respFmtGot, r) }).ToObject(r.vm)
		for _, method := range []string{"get", "post", "put", "head", "patch", "delete"} {
			got.Set(method, func(req, opts goja.Value) *goja.Promise {
				return fetch(req, opts, method, respFmtGot, r)
			})
		}
		module.Get("exports").ToObject(r.vm).Set("default", got)
	},
	"@/utils/cache": func(module *goja.Object, r *requireModule) {
		o := r.vm.NewObject()
		o.Set("tryGet", func(key string, f func() goja.Value, maxAge *int, ex *bool) *goja.Promise {
			promise, resolve, reject := r.vm.NewPromise()
			go func() {
				ttl := contentExpire
				if maxAge != nil {
					ttl = time.Duration(*maxAge) * time.Second
				}
				val, err := r.r.s.TryGet(storage.CONTENT, key, ttl, ex == nil || *ex, func() ([]byte, error) {
					var w wait
					w.Add(1)
					r.jobs <- func() { w.ThenDone(r.vm, f()) }
					w.Wait()
					if w.Err != nil {
						return nil, w.Err
					}
					return json.Marshal(w.Value)
				})
				var data any
				if err == nil {
					err = json.Unmarshal(utils.StringToBytes(val), &data)
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
		module.Get("exports").ToObject(r.vm).Set("default", o)
	},
}

var errNoSuchModule = errors.New("no such module")

func (r *requireModule) require(p string) (goja.Value, error) {
	p = r.resolve(p)
	if module, ok := r.modules[p]; ok {
		return module, nil
	}

	var module *goja.Object
	if strings.HasPrefix(p, "/") {
		src, err := r.r.route(p)
		if err != nil {
			return nil, err
		}
		module, err = loadIIFEModule(p, src, r.vm)
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

	} else if ldr, ok := native[p]; ok {
		module = r.vm.NewObject()
		module.Set("exports", r.vm.NewObject())
		ldr(module, r)

	} else if name, found := strings.CutPrefix(p, "@/"); found {
		if name == "utils/render" {
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

		} else if basename, found := strings.CutPrefix(name, "errors/types/"); found {
			var name string
			for _, word := range strings.Split(basename, "-") {
				name += strings.ToUpper(word[:1]) + word[1:]
			}
			name += "Error"
			val, err := r.vm.RunScript(p, fmt.Sprintf("(class extends Error{name='%s'})", name))
			if err != nil {
				return nil, err
			}
			exports := r.vm.NewObject()
			exports.Set("default", val)
			module = r.vm.NewObject()
			module.Set("exports", exports)

		} else {
			src, err := lib.ReadFile(name + ".js")
			if err == nil {
				module, err = loadIIFEModule(p, utils.BytesToString(src), r.vm)
				if err != nil {
					return nil, err
				}
			} else if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
		}

	} else {
		src, err := third_party.ReadFile(path.Join("third_party", p+".js"))
		if err == nil {
			module, err = loadIIFEModule(p, utils.BytesToString(src), r.vm)
			if err != nil {
				return nil, err
			}
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	if module != nil {
		module := module.Get("exports")
		if o := module.ToObject(r.vm); o.Get("default") == nil {
			o.Set("default", module)
		}
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

func loadIIFEModule(name, src string, vm *goja.Runtime) (*goja.Object, error) {
	f, err := vm.RunScript(name, src)
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
	if req.ExportType().Kind() == reflect.String || r.vm.InstanceOf(req, r.vm.Get("URL").ToObject(r.vm)) {
		if opts == nil || !opts.ToBoolean() {
			opts = r.vm.NewObject()
		}
		opts.ToObject(r.vm).Set("url", req.String())
	} else {
		opts = req
	}
	options := new(options)
	err := r.vm.ExportTo(opts, options)
	if err != nil {
		reject(err)
		return promise
	}
	if method != "" {
		options.Method = method
	}

	go func() {
		resp, err := r.r.fetch(options, respFmt)
		r.jobs <- func() {
			if err == nil {
				if options.ResponseType == "buffer" || options.ResponseType == "arrayBuffer" {
					vm := r.vm
					switch respFmt {
					case respFmtOfetch:
						resp = buffer.WrapBytes(vm, resp.([]byte))
					case respFmtOfetchRaw:
						r := resp.(*response)
						r.Data2 = buffer.WrapBytes(vm, r.Data2.([]byte))
						resp = r
					case respFmtGot:
						r := resp.(*response)
						r.Body = buffer.WrapBytes(vm, r.Body.([]byte))
						r.Data = buffer.WrapBytes(vm, r.Data.([]byte))
						resp = r
					}
				}
				resolve(resp)
			} else {
				reject(err)
			}
		}
	}()
	return promise
}
