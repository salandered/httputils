package httputils

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

const maxLoggedPayload = 512

// marshals then truncates, unless v renders itself
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
				Resolve loops LogValue() => the dispatch below runs HERE
*/
func (t truncatedValue) LogValue() slog.Value {
	// If value satisfies [slog.LogValuer] it knowns how to render itself
	if lv, ok := t.v.(slog.LogValuer); ok {
		return lv.LogValue()
	}
	raw, err := json.Marshal(t.v)
	if err != nil {
		return slog.StringValue(truncatePayload(fmt.Appendf(nil, "%+v", t.v)))
	}
	return slog.StringValue(truncatePayload(raw))
}

func truncatePayload(rawJSON []byte) string {
	if len(rawJSON) <= maxLoggedPayload {
		return string(rawJSON)
	}
	// json.Marshal emits raw UTF-8, so just a cut might split a rune
	return strings.ToValidUTF8(string(rawJSON[:maxLoggedPayload]), "") + "..."
}
