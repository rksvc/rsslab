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

func (w *wait) Await(vm *goja.Runtime, promise goja.Value, export bool) {
	then, _ := goja.AssertFunction(promise.ToObject(vm).Get("then"))
	_, err := then(promise, vm.ToValue(func(value goja.Value) {
		if export {
			w.Value = value.Export()
		} else {
			w.Value = value
		}
		w.Done()
	}), vm.ToValue(func(reason *goja.Object) {
		if stack := reason.Get("stack"); stack != nil && !goja.IsUndefined(stack) {
			w.Err = errors.New(stack.String())
		} else if err, ok := reason.Export().(error); ok {
			w.Err = err
		} else {
			w.Err = errors.New(reason.String())
		}
		w.Done()
	}))
	if err != nil {
		w.Err = err
		w.Done()
	}
}

func (r *RSSHub) handle(sourcePath string, ctx *ctx) (data any, err error) {
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
					h := resp.Headers.(http.Header)
					resp.Headers = vm.NewDynamicObject(&headers{map[string]goja.Value{
						"getSetCookie": vm.ToValue(func() []string {
							return h.Values("Set-Cookie")
						}),
						"get": vm.ToValue(func(key string) string {
							return h.Get(key)
						}),
					}})
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
					loop.RunOnLoop(func(vm *goja.Runtime) { w.Await(vm, vm.ToValue(f()), true) })
					w.Wait()
					return w.Value, w.Err
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
		w.Await(vm, v, false)
	})
	w.Wait()
	if w.Err != nil {
		return nil, w.Err
	}
	w.Add(1)
	loop.RunOnLoop(func(vm *goja.Runtime) {
		defer w.Done()
		value := w.Value.(goja.Value)
		items := value.ToObject(vm).Get("item")
		if goja.IsUndefined(items) || goja.IsNull(items) {
			data = value.Export()
			return
		}
		date := vm.Get("Date")
		e := vm.Try(func() {
			vm.ForOf(items, func(item goja.Value) bool {
				o := item.ToObject(vm)
				for _, key := range []string{"pubDate", "updated"} {
					v := o.Get(key)
					if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
						continue
					}
					v, err := vm.New(date, v)
					if err != nil {
						panic(err)
					}
					f, _ := goja.AssertFunction(v.ToObject(vm).Get("toISOString"))
					v, err = f(v)
					if err != nil {
						panic(err)
					}
					o.Set(key, v)
				}
				return true
			})
		})
		if e == nil {
			data = value.Export()
		} else {
			err = e
		}
	})
	w.Wait()
	return data, err
}
