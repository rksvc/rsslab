package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"rsslab/server"
	"rsslab/storage"
)

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
	storage, err := storage.New(database)
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(server.New(storage).Start(addr))
}
