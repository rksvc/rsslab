package main

import (
	"os"
	"path"

	"github.com/evanw/esbuild/pkg/api"
)

func main() {
	for _, pkg := range []struct{ entryPoint, outfile string }{
		{"node_modules/cheerio", "third_party/cheerio.js"},
		{"node_modules/crypto-js", "third_party/crypto-js.js"},
		{"node_modules/lz-string", "third_party/lz-string.js"},
		{"node_modules/markdown-it", "third_party/markdown-it.js"},
		{"node_modules/query-string", "third_party/query-string.js"},
		{"node_modules/query-string", "third_party/querystring.js"},
		{"lib/errors/types/config-not-found.ts", "lib/errors/types/config-not-found.js"},
		{"lib/errors/types/invalid-parameter.ts", "lib/errors/types/invalid-parameter.js"},
		{"lib/errors/types/not-found.ts", "lib/errors/types/not-found.js"},
		{"lib/errors/types/reject.ts", "lib/errors/types/reject.js"},
		{"lib/errors/types/request-in-progress.ts", "lib/errors/types/request-in-progress.js"},
		{"lib/utils/parse-date.ts", "lib/utils/parse-date.js"},
		{"lib/utils/readable-social.ts", "lib/utils/readable-social.js"},
		{"lib/utils/timezone.ts", "lib/utils/timezone.js"},
		{"lib/utils/valid-host.ts", "lib/utils/valid-host.js"},
	} {
		result := api.Build(api.BuildOptions{
			EntryPoints:       []string{path.Join("../deps/rsshub", pkg.entryPoint)},
			Outfile:           pkg.outfile,
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
		})
		if len(result.Errors) > 0 {
			os.Exit(1)
		}
	}
}
