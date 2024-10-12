package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rsslab/cache"
	"rsslab/rsshub"
	"rsslab/server"
	"rsslab/storage"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

var addr, redisUrl, database, routesUrl, srcUrl string
var cc *cache.Cache
var db *storage.Storage
var api *server.Server
var rssHub *rsshub.RSSHub

//go:embed dist
var assets embed.FS
var fsConfig = filesystem.Config{
	Root:       http.FS(assets),
	PathPrefix: "dist",
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	flag.StringVar(&addr, "addr", "127.0.0.1:9854", "address to run server on")
	flag.StringVar(&redisUrl, "redis", "", "redis `url` like redis://127.0.0.1:6379, omit to use in-memory cache")
	flag.StringVar(&database, "db", "", "storage file `path`")
	flag.StringVar(&routesUrl, "routes", "https://raw.githubusercontent.com/DIYgod/RSSHub/gh-pages/build/routes.json", "routes `url`")
	flag.StringVar(&srcUrl, "src", "https://raw.githubusercontent.com/DIYgod/RSSHub/master", "source code `url` prefix")
	flag.Parse()

	if redisUrl == "" {
		cc = cache.NewCache(cache.NewLRU())
	} else {
		cc = cache.NewCache(cache.NewRedis(redisUrl))
	}

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
	var err error
	db, err = storage.New(database)
	if err != nil {
		log.Fatal(err)
	}

	api = server.New(db)
	rssHub = rsshub.NewRSSHub(cc, routesUrl, srcUrl)

	app := engine()
	app.Use("/", filesystem.New(fsConfig))
	api.App.Store(app)
	go serve(app)
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
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	api.Register(app.Group("/api"))
	return app
}

func reload() bool {
	log.Printf("loading routes from %s", routesUrl)

	app := engine()
	err := rssHub.Register(app)
	if err != nil {
		log.Print(err)
		return false
	}
	rssHub.ClearCachedModules()
	app.Use("/", filesystem.New(fsConfig))

	oldApp := api.App.Swap(app).(*fiber.App)
	go func() {
		if err := oldApp.Shutdown(); err != nil {
			log.Print(err)
		}
		serve(app)
	}()
	return true
}

func serve(app *fiber.App) {
	host, port := addr, ""
	if i := strings.LastIndexByte(addr, ':'); i != -1 {
		host, port = addr[:i], addr[i+1:]
	}
	if host == "" {
		host = "0.0.0.0"
	}
	log.Printf("server started on http://%s:%s (%d handlers)", host, port, app.HandlersCount())
	err := app.Listen(addr)
	if err != nil {
		log.Fatal(err)
	}
}
