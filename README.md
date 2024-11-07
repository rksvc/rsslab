RSS reader with built-in feed generator.

## Build

```bash
pnpm install
pnpm build
go build -ldflags='-s -w' -tags=sqlite_foreign_keys -trimpath
```

## Credits

- [yarr](https://github.com/nkanaev/yarr/) for RSS reader.
- [errors](https://github.com/go-errors/errors) for storage module error handling.
