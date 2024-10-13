package rsshub

import (
	"crypto/md5"
	"embed"
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

//go:embed utils
var lib embed.FS

//go:embed third_party
var third_party embed.FS

var core = make(map[string]*goja.Program)
var native = map[string]moduleLoader{
	// Node.js modules
	"assert": func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("strict", func(value goja.Value, message goja.Value) error {
			if value.ToBoolean() {
				return nil
			}
			return errors.New(message.String())
		})
	},
	"path": func(vm *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").ToObject(vm)
		o.Set("join", func(elem ...string) string { return path.Join(elem...) })
		o.Set("dirname", func(p string) string { return path.Dir(p) })
	},
	"url": func(vm *goja.Runtime, module *goja.Object) {
		url.Require(vm, module)
		module.Get("exports").ToObject(vm).Set("fileURLToPath", func(url string) string { return url })
	},

	// RSSHub dependencies
	"dotenv/config": func(_ *goja.Runtime, _ *goja.Object) {},
	"ofetch":        func(_ *goja.Runtime, _ *goja.Object) {},

	// RSSHub source files
	"@/types": func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("ViewType", vm.NewObject())
	},
	"@/utils/md5": func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func(data string) string { return fmt.Sprintf("%x", md5.Sum(utils.StringToBytes(data))) })
	},
	"@/utils/rand-user-agent": func(_ *goja.Runtime, module *goja.Object) {
		module.Set("exports", func() string { return utils.USER_AGENT })
	},
	"@/utils/logger": func(vm *goja.Runtime, module *goja.Object) {
		o := module.Get("exports").ToObject(vm)
		for _, name := range []string{"debug", "info", "warn", "error", "http"} {
			o.Set(name, func() {})
		}
	},
	"@/utils/cache": func(vm *goja.Runtime, module *goja.Object) {
		module.Get("exports").ToObject(vm).Set("tryGet", vm.Get("$tryGet"))
	},
}

var errNoSuchModule = errors.New("no such module")

type moduleLoader func(*goja.Runtime, *goja.Object)

type requireModule struct {
	r       *RSSHub
	vm      *goja.Runtime
	modules map[string]goja.Value
}

func init() {
	for _, words := range [][]string{
		{"config", "not", "found"},
		{"invalid", "parameter"},
		{"not", "found"},
		{"reject"},
		{"request", "in", "progress"},
	} {
		path := "@/errors/types/" + strings.Join(words, "-")
		native[path] = func(vm *goja.Runtime, module *goja.Object) {
			var name string
			for _, word := range words {
				name += strings.ToUpper(word[:1]) + word[1:]
			}
			name += "Error"
			val, err := vm.RunScript(path, fmt.Sprintf("(class extends Error{name='%s'})", name))
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
		ldr(r.vm, module)

	} else if p == "@/utils/render" {
		v, err := r.require("art-template")
		if err != nil {
			return nil, err
		}
		art := v.ToObject(r.vm)
		render := r.vm.ToValue(func(filename string, content goja.Value) (goja.Value, error) {
			source, err := r.r.file(filename)
			if err != nil {
				return goja.Undefined(), err
			}
			render, _ := goja.AssertFunction(art.Get("render"))
			return render(goja.Undefined(), r.vm.ToValue(source), content, r.vm.ToValue(map[string]bool{
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
