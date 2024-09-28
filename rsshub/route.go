//go:generate go run ../cmd/bundle
package rsshub

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/buffer"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/process"
	"github.com/dop251/goja_nodejs/require"
	"github.com/dop251/goja_nodejs/url"
)

type ctx struct {
	Req req `json:"req"`
}

func (ctx *ctx) Set() {}

type req struct {
	Path string `json:"path"`

	queries map[string]string
	params  map[string]string
}

func (req *req) Param(key *string) any {
	if key == nil {
		return req.params
	} else if param, ok := req.params[*key]; ok {
		return param
	}
	return goja.Undefined()
}

func (req *req) Query(key *string) any {
	if key == nil {
		return req.queries
	} else if query, ok := req.queries[*key]; ok {
		return query
	}
	return goja.Undefined()
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
		if goja.IsUndefined(stack) {
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

func (r *RSSHub) handle(sourcePath string, ctx *ctx) (any, error) {
	loop := eventloop.NewEventLoop(eventloop.WithRegistry(r.registry.Load().(*require.Registry)))
	loop.Start()
	defer loop.Stop()
	w := newWait()
	loop.RunOnLoop(func(vm *goja.Runtime) {
		vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
		process.Enable(vm)
		url.Enable(vm)
		require.Require(vm, "url").ToObject(vm).Set("fileURLToPath", func(url string) string { return url })

		vm.Set("$fetch", func(opts map[string]any) *goja.Promise {
			promise, resolve, reject := vm.NewPromise()
			go func() {
				resp, err := r.fetch(opts)
				loop.RunOnLoop(func(vm *goja.Runtime) {
					if err != nil {
						reject(err)
						return
					}
					if b, ok := resp.Body.([]byte); ok {
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
		vm.Set("$tryGet", func(key string, f func() *goja.Promise, maxAge *int, ex *bool) *goja.Promise {
			promise, resolve, reject := vm.NewPromise()
			go func() {
				ttl := r.contentCacheTTL
				if maxAge != nil {
					ttl = time.Duration(*maxAge) * time.Second
				}
				v, err := r.cache.TryGet(key, ttl, ex == nil || *ex, func() (any, error) {
					w := newWait()
					loop.RunOnLoop(func(vm *goja.Runtime) { w.Await(vm, vm.ToValue(f())) })
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
					if err == nil {
						resolve(data)
					} else {
						reject(err)
					}
				})
			}()
			return promise
		})

		defer w.Done()
		var v goja.Value
		v, w.Err = vm.RunString(fmt.Sprintf("require('./%s').route.handler", sourcePath))
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
