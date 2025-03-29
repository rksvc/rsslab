package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"rsslab/server"
	"rsslab/storage"
	"rsslab/utils"

	"fyne.io/systray"
	"github.com/mattn/go-isatty"
	"github.com/pkg/browser"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func defaultConfigDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	dir = filepath.Join(dir, "rsslab")
	if err = os.MkdirAll(dir, 0755); err != nil {
		log.Fatal(err)
	}
	return dir
}

func main() {
	var addr, database, logFile string
	flag.StringVar(&addr, "addr", "127.0.0.1:9854", "address to run server on")
	flag.StringVar(&database, "db", "", "storage file `path`")
	flag.StringVar(&logFile, "log", "", "`path` to log file")
	flag.Parse()

	var configDir string
	if logFile == "" && !isatty.IsTerminal(os.Stderr.Fd()) && !isatty.IsCygwinTerminal(os.Stderr.Fd()) {
		if configDir == "" {
			configDir = defaultConfigDir()
		}
		logFile = filepath.Join(configDir, "rsslab.log")
	}
	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		log.SetOutput(f)
	}
	if database == "" {
		if configDir == "" {
			configDir = defaultConfigDir()
		}
		database = filepath.Join(configDir, "rsslab.db")
	}
	storage, err := storage.New(database)
	if err != nil {
		log.Fatal(err)
	}
	srv := server.New(storage)

	systray.Run(func() {
		systray.SetIcon(utils.Icon)
		systray.SetTitle("RSSLab")
		systray.SetTooltip("RSSLab")

		menuOpen := systray.AddMenuItem("Open", "")
		systray.AddSeparator()
		menuQuit := systray.AddMenuItem("Quit", "")
		go func() {
			for {
				select {
				case <-menuOpen.ClickedCh:
					if err := browser.OpenURL(srv.URL); err != nil {
						log.Print(err)
					}
				case <-menuQuit.ClickedCh:
					systray.Quit()
				}
			}
		}()

		log.Fatal(srv.Start(addr))
	}, nil)
}
