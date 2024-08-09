package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"rsslab/cache"
	"rsslab/rsshub"
	"rsslab/server"
	"rsslab/storage"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/filesystem"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

var noCache bool
var addr, redisUrl, database, routesUrl, srcUrl string
var cc cache.ICache
var db *storage.Storage
var api *server.Server
var srv atomic.Value
var reloading atomic.Bool

const RSSHUB_PATH = "/rsshub"

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.BoolVar(&noCache, "no-cache", false, "do not use cache")
	flag.StringVar(&addr, "addr", "127.0.0.1:9854", "address to run server on")
	flag.StringVar(&redisUrl, "redis", "", "redis `url` like redis://127.0.0.1:6379, omit to use in-memory cache")
	flag.StringVar(&database, "db", "", "storage file `path`")
	flag.StringVar(&routesUrl, "routes", "https://raw.githubusercontent.com/DIYgod/RSSHub/gh-pages/build/routes.json", "routes `url`")
	flag.StringVar(&srcUrl, "src", "https://unpkg.com/rsshub", "source code `url` prefix")
	flag.Parse()
	if database == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		dir = filepath.Join(dir, "rsslab")
		if err = os.MkdirAll(dir, 0755); err != nil {
			log.Fatal(err)
		}
		database = filepath.Join(dir, "storage.db")
	}

	if noCache {
		cc = cache.NewDisabled()
	} else if redisUrl == "" {
		cc = cache.NewLRU()
	} else {
		cc = cache.NewRedis(redisUrl)
	}

	var err error
	db, err = storage.New(database)
	if err != nil {
		log.Fatal(err)
	}

	rsshubBaseUrl, err := url.Parse("http://" + addr + RSSHUB_PATH)
	if err != nil {
		log.Fatal(err)
	}
	api = server.New(db, rsshubBaseUrl)

	app := engine()
	dist, _ := fs.Sub(dist, "dist")
	app.Use("/", filesystem.New(filesystem.Config{Root: dist}))
	srv.Store(app)

	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatal(err)
		}
	}()
	for !reload() {
		time.Sleep(10 * time.Second)
	}
	api.Start()
	for {
		time.Sleep(6 * time.Hour)
		reload()
	}
}

//go:embed dist
var dist embed.FS

func engine() *fiber.App {
	app := fiber.New()
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(compress.New())
	api.Register(app.Group("/api"))
	app.All("/reload", func(c fiber.Ctx) error {
		if reloading.Load() {
			return c.SendStatus(http.StatusConflict)
		}
		go reload()
		return c.SendStatus(http.StatusOK)
	})
	return app
}

func reload() bool {
	reloading.Store(true)
	defer reloading.Store(false)
	log.Printf("loading routes from %s", routesUrl)

	rsshub, err := rsshub.NewRSSHub(cc, routesUrl, srcUrl)
	if err != nil {
		log.Print(err)
		return false
	}
	app := engine()
	rsshub.Register(app.Group(RSSHUB_PATH))
	dist, _ := fs.Sub(dist, "dist")
	app.Use("/", filesystem.New(filesystem.Config{Root: dist}))

	if err := srv.Swap(app).(*fiber.App).ShutdownWithTimeout(5 * time.Second); err != nil {
		log.Print(err)
	}
	go func() {
		if err := app.Listen(addr); err != nil {
			log.Fatal(err)
		}
	}()
	return true
}
