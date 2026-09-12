# httputils

Small dependency, bad name. Please ignore that this repo exists.

JSON request/response utils for `net/http` handlers. stdlib only.

## Logging

Every function logs through `slog.Default()`.
The importing project owns the handler.

`ReadJSON` logs the decoded body and `WriteJSON` logs the response payload, both at Debug level.

A type holding a secret might implement `slog.LogValuer`:

```go
func (r CreateTokenReq) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("nickname", r.Nickname),
		slog.String("password", "[redacted]"),
	)
}
```

## Dev

### Test

```sh
make audit
go test -race ./...
```

### Release

Check CI is ok.

```sh
git tag --list
git tag -a v0.x.0 -m "v0.x.0"
git push origin v0.x.0
```
