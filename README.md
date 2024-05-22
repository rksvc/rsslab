Feed reader with built-in [RSSHub](https://github.com/DIYgod/RSSHub) service which lets you subscribe to feeds using `rsshub` protocol URLs like `rsshub:github/issue/DIYgod/RSSHub`. It fetches the source code of RSSHub on demand and runs it locally in an embedded JavaScript engine to generate feeds.

Routes that require Puppeteer are not currently supported.

## Usage

Execute the binary and open http://127.0.0.1:9854 in your browser. The RSSHub integration is not ready until you see the `registered ... routes` output on the console.

Additionally, you can view the generated feed in `http://127.0.0.1:9854/rsshub/{route}` like http://127.0.0.1:9854/rsshub/github/issue/DIYgod/RSSHub.

Set `https_proxy` environment variable to use the proxy.

Alternatives for some command line argument values:

- `-src`
  - `https://unpkg.com/rsshub` (default)
  - `https://cdn.jsdelivr.net/npm/rsshub`
  - `https://raw.githubusercontent.com/DIYgod/RSSHub/master`
- `-routes`
  - `https://raw.githubusercontent.com/DIYgod/RSSHub/gh-pages/build/routes.json` (default)
  - `https://rsshub.app/api/namespace`

## Build

Install `libsqlite3-dev`. Run:

```bash
git submodule update --init
cd ./deps/rsshub
PUPPETEER_SKIP_DOWNLOAD=1 pnpm i
cd ../..
go generate ./...
pnpm i
pnpm build
go build -ldflags='-s -w' -tags='sqlite_foreign_keys sqlite_fts5' -trimpath
```

## Credits

- [yarr](https://github.com/nkanaev/yarr/) for RSS reader.
- [simple](https://github.com/wangfenjin/simple) for SQLite FTS5 tokenizer.
