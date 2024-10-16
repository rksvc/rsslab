//go:generate go run ../cmd/bundle
package rsshub

import (
	_ "embed"
	"errors"
	"rsslab/utils"
	"sync"

	"github.com/dop251/goja"
)

//go:embed handler.js
var handler string
var handlerPrg = goja.MustCompile("", handler, false)

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

func (w *wait) Await(vm *goja.Runtime, promise goja.Value) {
	then, _ := goja.AssertFunction(promise.ToObject(vm).Get("then"))
	_, err := then(promise, vm.ToValue(func(value goja.Value) {
		w.Value = value.Export()
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

func (r *RSSHub) handle(path string, ctx *ctx) (any, error) {
	vm := goja.New()
	jobs := make(chan func())
	require := &requireModule{
		r:       r,
		vm:      vm,
		jobs:    jobs,
		modules: make(map[string]goja.Value),
	}
	go func() {
		for job := range jobs {
			job()
		}
	}()

	vm.SetFieldNameMapper(goja.TagFieldNameMapper("json", true))
	vm.Set("require", require.require)

	process := vm.NewObject()
	process.Set("env", utils.Env)
	vm.Set("process", process)

	buffer, err := require.require("buffer")
	if err != nil {
		return nil, err
	}
	vm.Set("Buffer", buffer.ToObject(vm).Get("Buffer"))

	url, err := require.require("url")
	if err != nil {
		return nil, err
	}
	exports := url.ToObject(vm)
	vm.Set("URL", exports.Get("URL"))
	vm.Set("URLSearchParams", exports.Get("URLSearchParams"))

	val, err := vm.RunProgram(handlerPrg)
	if err != nil {
		return nil, err
	}
	handler, _ := goja.AssertFunction(val)
	var w wait
	w.Add(1)
	jobs <- func() {
		defer w.Done()
		val, w.Err = handler(goja.Undefined(), vm.ToValue(path), vm.ToValue(ctx))
		if w.Err != nil {
			return
		}
		w.Add(1)
		w.Await(vm, val)
	}
	w.Wait()
	close(jobs)
	return w.Value, w.Err
}
