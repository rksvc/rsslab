package main

import (
	"io"
	"net/http"
	"os"
	"rsslab/utils"
	"sync"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	files := []string{
		"helpers",
		"parse-date",
		"readable-social",
		"timezone",
		"valid-host",
	}
	cleanup := func() {
		for _, file := range files {
			os.RemoveAll(file + ".ts")
		}
	}
	defer cleanup()
	var wg sync.WaitGroup
	wg.Add(len(files))
	for _, file := range files {
		go func() {
			defer wg.Done()
			resp, err := http.Get("https://raw.githubusercontent.com/DIYgod/RSSHub/master/lib/utils/" + file + ".ts")
			if err != nil {
				panic(err)
			}
			defer resp.Body.Close()
			if utils.IsErrorResponse(resp.StatusCode) {
				panic(utils.ResponseError(resp))
			}
			f, err := os.Create(file + ".ts")
			if err != nil {
				panic(err)
			}
			defer f.Close()
			_, err = io.Copy(f, resp.Body)
			if err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()

	opts := api.BuildOptions{
		Platform:          api.PlatformNode,
		Sourcemap:         api.SourceMapInline,
		SourcesContent:    api.SourcesContentExclude,
		Target:            api.ES2023,
		Supported:         utils.SupportedSyntaxFeatures,
		LogLevel:          api.LogLevelInfo,
		Banner:            map[string]string{"js": utils.IIFE_PREFIX},
		Footer:            map[string]string{"js": utils.IIFE_SUFFIX},
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
		EntryPointsAdvanced: []api.EntryPoint{
			{InputPath: "../node_modules/art-template/lib", OutputPath: "third_party/art-template"},
			{InputPath: "../node_modules/cheerio/dist/browser", OutputPath: "third_party/cheerio"},
			{InputPath: "../node_modules/lz-string", OutputPath: "third_party/lz-string"},
			{InputPath: "../node_modules/markdown-it", OutputPath: "third_party/markdown-it"},
			{InputPath: "../node_modules/query-string", OutputPath: "third_party/query-string"},
			{InputPath: "../node_modules/query-string", OutputPath: "third_party/querystring"},
			{InputPath: "../node_modules/sanitize-html", OutputPath: "third_party/sanitize-html"},
		},
		Outdir:   ".",
		External: []string{"html-minifier"},
	}
	for _, file := range files {
		opts.EntryPointsAdvanced = append(opts.EntryPointsAdvanced, api.EntryPoint{
			InputPath:  file + ".ts",
			OutputPath: "utils/" + file,
		})
	}
	result := api.Build(opts)
	if len(result.Errors) > 0 {
		cleanup()
		os.Exit(1)
	}
}
