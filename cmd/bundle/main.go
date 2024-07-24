package main

import (
	"os"
	"path"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	for _, pkg := range []struct {
		entryPoint, outfile string
		external            []string
	}{
		{"node_modules/art-template/lib", "third_party/art-template.js", []string{"html-minifier"}},
		{"node_modules/cheerio", "third_party/cheerio.js", nil},
		{"node_modules/crypto-js", "third_party/crypto-js.js", nil},
		{"node_modules/lz-string", "third_party/lz-string.js", nil},
		{"node_modules/markdown-it", "third_party/markdown-it.js", nil},
		{"node_modules/query-string", "third_party/query-string.js", nil},
		{"node_modules/query-string", "third_party/querystring.js", nil},
		{"lib/types.ts", "lib/types.js", nil},
		{"lib/errors/types/config-not-found.ts", "lib/errors/types/config-not-found.js", nil},
		{"lib/errors/types/invalid-parameter.ts", "lib/errors/types/invalid-parameter.js", nil},
		{"lib/errors/types/not-found.ts", "lib/errors/types/not-found.js", nil},
		{"lib/errors/types/reject.ts", "lib/errors/types/reject.js", nil},
		{"lib/errors/types/request-in-progress.ts", "lib/errors/types/request-in-progress.js", nil},
		{"lib/utils/helpers.ts", "lib/utils/helpers.js", nil},
		{"lib/utils/parse-date.ts", "lib/utils/parse-date.js", nil},
		{"lib/utils/readable-social.ts", "lib/utils/readable-social.js", nil},
		{"lib/utils/timezone.ts", "lib/utils/timezone.js", nil},
		{"lib/utils/valid-host.ts", "lib/utils/valid-host.js", nil},
	} {
		result := api.Build(api.BuildOptions{
			EntryPoints:       []string{path.Join("../deps/rsshub", pkg.entryPoint)},
			Outfile:           pkg.outfile,
			Platform:          api.PlatformNode,
			Sourcemap:         api.SourceMapInline,
			SourcesContent:    api.SourcesContentExclude,
			Target:            api.ES2017,
			LogLevel:          api.LogLevelInfo,
			External:          pkg.external,
			Bundle:            true,
			Write:             true,
			MinifyWhitespace:  true,
			MinifySyntax:      true,
			MinifyIdentifiers: true,
		})
		if len(result.Errors) > 0 {
			os.Exit(1)
		}
	}
}
