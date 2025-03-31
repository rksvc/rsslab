RSS reader with built-in feed generator.

## Build

```sh
pnpm install
pnpm build
go build -tags=sqlite_foreign_keys -trimpath -ldflags='-s -w' # append ' -H=windowsgui' to -ldflags on Windows
```

## Credits

- [yarr](https://github.com/nkanaev/yarr/) for RSS reader.
- [fluentui-emoji](https://github.com/microsoft/fluentui-emoji) for icon.
