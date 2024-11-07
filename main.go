package main

import (
	"embed"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"rsslab/server"
	"rsslab/storage"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

//go:embed dist
var assets embed.FS

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	var addr, database string
	flag.StringVar(&addr, "addr", "127.0.0.1:9854", "address to run server on")
	flag.StringVar(&database, "db", "", "storage file `path`")
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
		database = filepath.Join(dir, "rsslab.db")
	}
	s, err := storage.New(database)
	if err != nil {
		log.Fatal(err)
	}

	api := server.New(s)
	api.Start()

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(recover.New(recover.Config{EnableStackTrace: true}))
	api.Register(app.Group("/api"))
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(assets),
		PathPrefix: "dist",
	}))

	host, port := addr, ""
	if i := strings.LastIndexByte(addr, ':'); i != -1 {
		host, port = addr[:i], addr[i+1:]
	}
	if host == "" {
		host = "0.0.0.0"
	}
	log.Printf("server started on http://%s:%s", host, port)
	log.Fatal(app.Listen(addr))
}
