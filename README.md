Feed reader built on [nkanaev/yarr](https://github.com/nkanaev/yarr) with built-in RSSHub service which lets you subscribe to feeds using `rsshub` protocol URLs like `rsshub:github/issue/DIYgod/RSSHub`. It fetches the source code on demand from the npm Registry and runs it locally in an embedded JavaScript runtime to generate feeds.

Routes that require Puppeteer, art-template, etc. are not currently supported.

## Usage

Execute the binary and open http://127.0.0.1:9854 in your browser. The RSSHub integration is not ready until you see the `registered ... routes` output on the console.

Additionally, you can view the generated feed in `http://127.0.0.1:9854/rsshub/{route}` like http://127.0.0.1:9854/rsshub/github/issue/DIYgod/RSSHub.

Alternatives for some command line argument values:

- `-src`
  - `https://unpkg.com/rsshub` (default)
  - `https://registry.npmmirror.com/rsshub/latest/files` (recommended for users in Chinese mainland)
  - `https://cdn.jsdelivr.net/npm/rsshub`
  - `https://raw.githubusercontent.com/DIYgod/RSSHub/master`
- `-routes`
  - `https://raw.githubusercontent.com/DIYgod/RSSHub/gh-pages/build/routes.json` (default)
  - `https://rsshub.app/api/namespace`

Set `https_proxy` environment variable to use the proxy.

## Build

```bash
git submodule update --init
cd ./deps/rsshub
PUPPETEER_SKIP_DOWNLOAD=1 pnpm i
cd ../..
go generate ./...
go build -buildmode=exe -ldflags='-s -w' -tags='sqlite_foreign_keys sqlite_fts5' -trimpath
```

## Compare with yarr

The following features have been temporarily removed since I don't want to maintain them:

- authentication
- resizing feed list and item list
- discovering feed
- Fever API support
- 'Read Here'

, and it differs in the following ways:

- deletes feeds when deleting folders

, and has the following additional features:

- uses [siyuan tokenizer](https://github.com/siyuan-note/sqlite-fts5-siyuan-tokenizer) for better CJK text search which makes English text search worse
- supports editing feed link
- supports refreshing selected feeds
- supports refreshing feeds with errors
- shows 'Mark All Read' button when no filter applied
