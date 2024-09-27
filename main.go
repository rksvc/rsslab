package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
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
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

var noCache bool
var addr, redisUrl, database, routesUrl, srcUrl string
var cc *cache.Cache
var db *storage.Storage
var api *server.Server
var rssHub *rsshub.RSSHub
var srv atomic.Value

//go:embed dist
var dist embed.FS
var assets fs.FS

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	assets, _ = fs.Sub(dist, "dist")
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

	var c cache.ICache
	if noCache {
		c = cache.NewDisabled()
	} else if redisUrl == "" {
		c = cache.NewLRU()
	} else {
		c = cache.NewRedis(redisUrl)
	}
	cc = cache.NewCache(c)

	var err error
	db, err = storage.New(database)
	if err != nil {
		log.Fatal(err)
	}

	app := engine()
	app.Use("/", static.New("", static.Config{FS: assets}))
	api = server.New(db)
	api.App.Store(app)
	srv.Store(app)
	rssHub = rsshub.NewRSSHub(cc, routesUrl, srcUrl)

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

func engine() *fiber.App {
	app := fiber.New()
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	app.Use(compress.New())
	api.Register(app.Group("/api"))
	return app
}

func reload() bool {
	log.Printf("loading routes from %s", routesUrl)

	if err := rssHub.LoadRoutes(); err != nil {
		log.Print(err)
		return false
	}
	app := engine()
	rssHub.Register(app)
	app.Use("/", static.New("", static.Config{FS: assets}))

	api.App.Store(app)
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
