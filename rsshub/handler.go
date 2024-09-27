//go:generate go run ../cmd/bundle
package rsshub

import (
	"crypto/md5"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"rsslab/utils"
	"strings"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed utils
var lib embed.FS

//go:embed third_party
var third_party embed.FS

//go:embed polyfill.js
var polyfill string

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
		p := module.Get("exports").ToObject(vm)
		p.Set("join", func(elem ...string) string { return path.Join(elem...) })
		p.Set("dirname", func(p string) string { return path.Dir(p) })
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
}

func (r *RSSHub) sourceLoader(workingDirectory string) func(string) ([]byte, error) {
	const NODE_MODULES = "node_modules/"
	return func(filename string) ([]byte, error) {
		name := strings.ReplaceAll(filename, NODE_MODULES, "")

		const ROOT = "@/"
		if i := strings.LastIndex(name, ROOT); i != -1 {
			name := name[i+len(ROOT):]

			if name == "config" {
				return r.route("lib/config.ts")
			}

			if name, found := strings.CutPrefix(name, "errors/types/"); found {
				words := strings.Split(name, "-")
				var name string
				caser := cases.Title(language.AmericanEnglish)
				for _, word := range words {
					name += caser.String(word)
				}
				name += "Error"
				return []byte(fmt.Sprintf("module.exports=class %s extends Error{name='%s'}", name, name)), nil
			}

			data, err := lib.ReadFile(name + ".js")
			if err != nil {
				return nil, fmt.Errorf("require %s: %s", ROOT+name, require.ModuleFileDoesNotExistError)
			}
			return data, nil
		}

		if i := strings.LastIndex(filename, NODE_MODULES); i != -1 {
			data, err := third_party.ReadFile(path.Join("third_party", filename[i+len(NODE_MODULES):]+".js"))
			if err == nil {
				return data, nil
			}
		}

		if name == "tglib/channel" {
			return nil, nil
		}
		return r.route(path.Join("lib/routes", workingDirectory, name+".ts"))
	}
}

type wait struct {
	sync.WaitGroup
	Value any
	Err   error
}

func newWait() *wait {
	var w wait
	w.Add(1)
	return &w
}

func (w *wait) Result() (any, error) {
	if err, ok := w.Err.(*goja.Exception); ok {
		w.Err = errors.New(err.String())
	}
	return w.Value, w.Err
}

func (w *wait) Await(vm *goja.Runtime, promise goja.Value) {
	then, _ := goja.AssertFunction(promise.ToObject(vm).Get("then"))
	_, err := then(promise, vm.ToValue(func(value goja.Value) {
		w.Value = value.Export()
		w.Done()
	}), vm.ToValue(func(reason *goja.Object) {
		stack := reason.Get("stack")
		if stack == nil || goja.IsUndefined(stack) {
			if err, ok := reason.Export().(error); ok {
				w.Err = err
			} else {
				w.Err = errors.New(reason.String())
			}
		} else {
			w.Err = errors.New(stack.String())
		}
		w.Done()
	}))
	if err != nil {
		w.Err = err
		w.Done()
	}
}

func (r *RSSHub) Data(namespace, location string, ctx *Ctx) (any, error) {
	workingDirectory := path.Dir(path.Join(namespace, location))
	registry := require.NewRegistryWithLoader(r.sourceLoader(workingDirectory))
	loop := eventloop.NewEventLoop(eventloop.WithRegistry(registry))
	registry.RegisterNativeModule("@/utils/render", func(vm *goja.Runtime, module *goja.Object) {
		art := require.Require(vm, "art-template").ToObject(vm)
		render := vm.ToValue(func(filename string, content goja.Value) (goja.Value, error) {
			source, err := r.file(filename)
			if err != nil {
				return goja.Undefined(), err
			}
			render, _ := goja.AssertFunction(art.Get("render"))
			return render(goja.Undefined(), vm.ToValue(string(source)), content, vm.ToValue(map[string]any{
				"debug":    false,
				"minimize": false,
			}))
		}).ToObject(vm)
		err := render.Set("defaults", art.Get("defaults"))
		if err != nil {
			panic(err)
		}
		module.Get("exports").ToObject(vm).Set("art", render)
	})
	registry.RegisterNativeModule("@/utils/cache", func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("tryGet", func(key string, f func() *goja.Promise, maxAge *int, ex *bool) *goja.Promise {
			promise, resolve, reject := vm.NewPromise()
			go func() {
				ttl := r.contentCacheTTL
				if maxAge != nil {
					ttl = time.Duration(*maxAge) * time.Second
				}
				v, err := r.contentCache.TryGet(key, ttl, ex == nil || *ex, func() (result any, err error) {
					w := newWait()
					loop.RunOnLoop(func(*goja.Runtime) { w.Await(vm, vm.ToValue(f())) })
					w.Wait()
					return w.Result()
				})
				var data any
				if b, ok := v.([]byte); !ok {
					data = v
				} else if json.Unmarshal(b, &data) != nil {
					data = string(b)
				}
				loop.RunOnLoop(func(*goja.Runtime) {
					if err != nil {
						reject(err)
						return
					}
					resolve(data)
				})
			}()
			return promise
		})
	})

	w := newWait()
	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer w.Done()
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		url.Enable(vm)
		require.Require(vm, "url").ToObject(vm).Set("fileURLToPath", func(url string) string { return url })
		_, w.Err = vm.RunString(polyfill)
		if w.Err != nil {
			return
		}

		vm.Set("process", map[string]any{"env": utils.Env})
		vm.Set("$fetch", func(opts map[string]any) *goja.Promise {
			promise, resolve, reject := vm.NewPromise()
			go func() {
				resp, err := r.fetch(opts)
				loop.RunOnLoop(func(vm *goja.Runtime) {
					if err != nil {
						reject(err)
						return
					} else if b, ok := resp.Body.([]byte); ok {
						resp.Body = buffer.WrapBytes(vm, b)
						resp.Data = resp.Body
						resp.Data2 = resp.Body
					}
					resp.Headers = vm.NewDynamicObject(&headers{
						h:  resp.Headers.(http.Header),
						vm: vm,
					})
					resolve(resp)
				})
			}()
			return promise
		})

		var v goja.Value
		v, w.Err = vm.RunString(fmt.Sprintf("require('./%s').route.handler", path.Base(location)))
		if w.Err != nil {
			return
		}
		handler, _ := goja.AssertFunction(v)
		v, w.Err = handler(goja.Undefined(), vm.ToValue(ctx))
		if w.Err != nil {
			return
		}
		w.Add(1)
		w.Await(vm, v)
	})

	w.Wait()
	return w.Result()
}
