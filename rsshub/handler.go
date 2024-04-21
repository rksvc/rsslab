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

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

//go:embed lib
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
	require.RegisterNativeModule("dotenv/config", func(_ *goja.Runtime, _ *goja.Object) {})
	require.RegisterNativeModule("ofetch", func(_ *goja.Runtime, _ *goja.Object) {})
	require.RegisterNativeModule("@/utils/md5", func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func(data string) string { return fmt.Sprintf("%x", md5.Sum([]byte(data))) })
	})
	require.RegisterNativeModule("@/utils/rand-user-agent", func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func() string { return utils.UserAgent })
	})
}

func (r *RSSHub) sourceLoader(workingDirectory string) func(string) ([]byte, error) {
	const NODE_MODULES = "node_modules/"
	return func(filename string) ([]byte, error) {
		filename = strings.TrimPrefix(filename, NODE_MODULES)

		if name, found := strings.CutPrefix(filename, "@/"); found {
			name = strings.ReplaceAll(name, NODE_MODULES, "")
			if name == "config" {
				return r.route("lib/config.ts")
			}

			name = path.Join("lib", name+".js")
			data, err := lib.ReadFile(name)
			if err != nil {
				return nil, fmt.Errorf("require %s: %s", name, require.ModuleFileDoesNotExistError)
			}
			return data, nil
		}

		data, err := third_party.ReadFile(path.Join("third_party", path.Base(filename)+".js"))
		if err == nil {
			return data, nil
		}

		return r.route(path.Join("lib/routes", workingDirectory, filename+".ts"))
	}
}

func errorWithFullStack(err error) error {
	if err, ok := err.(*goja.Exception); ok {
		return errors.New(err.String())
	}
	return err
}

func await(vm *goja.Runtime, promise goja.Value, result chan<- any) {
	then, _ := goja.AssertFunction(promise.ToObject(vm).Get("then"))
	_, err := then(promise, vm.ToValue(func(value goja.Value) {
		result <- value.Export()
	}), vm.ToValue(func(reason *goja.Object) {
		stack := reason.Get("stack")
		if stack == nil || goja.IsUndefined(stack) {
			if err, ok := reason.Export().(error); ok {
				result <- err
			} else {
				result <- errors.New(reason.String())
			}
		} else {
			result <- errors.New(stack.String())
		}
	}))
	if err != nil {
		result <- errorWithFullStack(err)
	}
}

func (r *RSSHub) Data(namespace, location string, ctx *Ctx) any {
	result := make(chan any)

	workingDirectory := path.Dir(path.Join(namespace, location))
	registry := require.NewRegistryWithLoader(r.sourceLoader(workingDirectory))
	loop := eventloop.NewEventLoop(eventloop.WithRegistry(registry))
	registry.RegisterNativeModule("@/utils/cache", func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("tryGet", func(key string, f func() *goja.Promise, _ any, ex *bool) *goja.Promise {
			promise, resolve, reject := vm.NewPromise()
			go func() {
				v, err := r.contentCache.TryGet(key, ex == nil || *ex, func() (any, error) {
					var result = make(chan any)
					loop.RunOnLoop(func(*goja.Runtime) {
						await(vm, vm.ToValue(f()), result)
					})
					v := <-result
					if err, ok := v.(error); ok {
						return nil, err
					}
					return v, nil
				})
				var data any = v
				if v, ok := v.([]byte); ok && json.Unmarshal(v, &data) != nil {
					data = v
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

	loop.Start()
	defer loop.Stop()
	loop.RunOnLoop(func(vm *goja.Runtime) {
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		url.Enable(vm)
		_, err := vm.RunString(polyfill)
		if err != nil {
			result <- errorWithFullStack(err)
			return
		}

		vm.Set("process", vm.ToValue(map[string]any{"env": utils.Env}))
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

		v, err := vm.RunString(fmt.Sprintf("require('./%s').route.handler", path.Base(location)))
		if err != nil {
			result <- errorWithFullStack(err)
			return
		}
		handler, _ := goja.AssertFunction(v)
		v, err = handler(goja.Undefined(), vm.ToValue(ctx))
		if err != nil {
			result <- errorWithFullStack(err)
			return
		}
		await(vm, v, result)
	})

	return <-result
}
