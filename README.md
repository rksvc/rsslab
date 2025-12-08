RSS reader with built-in feed builder which can transform websites with HTML/JSON content into RSS feeds using CSS selectors/[JSON paths](https://github.com/tidwall/gjson).

## Usage

Run the executable, click the system tray icon, select "Open".

## Build

```sh
pnpm install
pnpm bundle
go build -tags=sqlite_foreign_keys -trimpath -ldflags='-s -w' # append ' -H=windowsgui' to -ldflags on Windows
```

## Credits

- [yarr](https://github.com/nkanaev/yarr/) for RSS reader.
- [fluentui-emoji](https://github.com/microsoft/fluentui-emoji) for icon.
