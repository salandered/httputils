# httputils

Small dependency, bad name. Please ignore that this repo exists.

JSON request/response utils for `net/http` handlers. stdlib only.

```go
import "github.com/salandered/httputils/httputils"
```

```go
func ParseIntQuery(req *http.Request, name string, def, min, max int64) (int64, error)
func ReadJSON(w http.ResponseWriter, req *http.Request, dst any, maxBytes int64) error
func WriteJSON(ctx context.Context, w http.ResponseWriter, statusCode int, data any)
func WriteError(ctx context.Context, w http.ResponseWriter, err error, statusCode int)
```

`ReadJSON` bounds the body at `maxBytes` (which must be positive), rejects unknown fields, and
rejects anything past the first JSON object. `WriteError` writes `{"error": "..."}`, logs 4xx at
Warn and 5xx at Error, and replaces a 5xx message with `internal server error` so the cause stays
in the log only.

The 512-byte cap on a logged payload, the `{"error": ...}` envelope, and the Debug level of the
decode/send lines are hardcoded. `ParseIntQuery` names its bounds `min`/`max`, shadowing the
builtins.

## Logging

Every function logs through `slog.Default()`.
The importing project owns the handler.

The decoded request body and the sent response payload are logged at Debug, both truncated to 512
bytes. Truncation runs inside `LogValue`, which `slog` resolves only for a record it has already
decided to emit, so above Debug the body is never marshalled and the payload never copied. The
call site still pays for wrapping the value.
