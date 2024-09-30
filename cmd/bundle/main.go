package main

import (
	"os"
	"path"
	"rsslab/utils"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/go-resty/resty/v2"
)

func main() {
	opts := api.BuildOptions{
		Platform:          api.PlatformNode,
		Sourcemap:         api.SourceMapInline,
		SourcesContent:    api.SourcesContentExclude,
		Target:            api.ES2017,
		LogLevel:          api.LogLevelInfo,
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
	}

	for _, pkg := range []struct {
		path, outfile string
		external      []string
	}{
		{"art-template/lib", "art-template.js", []string{"html-minifier"}},
		{"cheerio/dist/browser", "cheerio.js", nil},
		{"crypto-js", "crypto-js.js", nil},
		{"lz-string", "lz-string.js", nil},
		{"markdown-it", "markdown-it.js", nil},
		{"query-string", "query-string.js", nil},
		{"query-string", "querystring.js", nil},
	} {
		opts := opts
		opts.EntryPoints = []string{path.Join("../node_modules", pkg.path)}
		opts.Outfile = path.Join("third_party", pkg.outfile)
		opts.External = pkg.external
		if len(api.Build(opts).Errors) > 0 {
			os.Exit(1)
		}
	}

	files := []struct {
		path, outfile string
		content       []byte
	}{
		{"utils/helpers.ts", "utils/helpers.js", nil},
		{"utils/parse-date.ts", "utils/parse-date.js", nil},
		{"utils/readable-social.ts", "utils/readable-social.js", nil},
		{"utils/timezone.ts", "utils/timezone.js", nil},
		{"utils/valid-host.ts", "utils/valid-host.js", nil},
	}
	var wg sync.WaitGroup
	wg.Add(len(files))
	client := resty.New()
	for i, file := range files {
		go func() {
			defer wg.Done()
			resp, err := client.R().Get("https://raw.githubusercontent.com/DIYgod/RSSHub/master/lib/" + file.path)
			if err != nil {
				panic(err)
			} else if resp.IsError() {
				panic(utils.ResponseError(resp.RawResponse))
			}
			files[i].content = resp.Body()
		}()
	}
	wg.Wait()
	f, err := os.CreateTemp(".", "*.ts")
	if err != nil {
		panic(err)
	}
	cleanup := func() {
		f.Close()
		os.Remove(f.Name())
	}
	defer cleanup()
	for _, file := range files {
		err := f.Truncate(0)
		if err != nil {
			panic(err)
		}
		_, err = f.WriteAt(file.content, 0)
		if err != nil {
			panic(err)
		}
		opts := opts
		opts.EntryPoints = []string{f.Name()}
		opts.Outfile = file.outfile
		if len(api.Build(opts).Errors) > 0 {
			cleanup()
			os.Exit(1)
		}
	}
}
