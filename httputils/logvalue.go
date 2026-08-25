package httputils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

const maxLoggedPayload = 512

// Resolved by the handler, so nothing below is paid for when the record is dropped.

// marshals then truncates
type truncatedValue struct {
	v any
}

/*
Allows lazily marshall and trancate.

slog.DebugContext(..., truncatedValue{dst})

	│
	├─ truncatedValue{dst} is built
	│
	└─ Logger.log(ctx, LevelDebug, msg, args...)
		│
		├─ if !l.Enabled(ctx, level) { return }        (logger.go:244)
		│       LOG_LEVEL=info stops HERE.
		│  ...
		|
		└─ Handler.Handle → appendAttr → Value.Resolve()   (handler.go:477)
				Resolve loops LogValue() => json.Marshal + truncate run HERE
*/
func (t truncatedValue) LogValue() slog.Value {
	raw, err := json.Marshal(t.v)
	if err != nil {
		return slog.StringValue(truncatePayload(fmt.Appendf(nil, "%+v", t.v)))
	}
	return slog.StringValue(truncatePayload(raw))
}

// already-marshalled JSON
type truncatedJSON []byte

func (t truncatedJSON) LogValue() slog.Value {
	return slog.StringValue(truncatePayload(t))
}

func truncatePayload(rawJSON []byte) string {
	if len(rawJSON) <= maxLoggedPayload {
		return string(rawJSON)
	}
	// json.Marshal emits raw UTF-8, so just a cut might split a rune
	return strings.ToValidUTF8(string(rawJSON[:maxLoggedPayload]), "") + "..."
}
