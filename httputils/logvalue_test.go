package httputils

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTruncatePayloadLeavesShortPayloadAlone(t *testing.T) {
	require.Equal(t, `{"a":1}`, truncatePayload([]byte(`{"a":1}`)))
}

func TestTruncatePayloadCutsAndMarksLongPayload(t *testing.T) {
	got := truncatePayload([]byte(strings.Repeat("a", maxLoggedPayload+10)))

	require.True(t, strings.HasSuffix(got, "..."))
	require.Len(t, got, maxLoggedPayload+3) // 3 - for three dots
}

func TestTruncatePayloadDropsRuneSplitByTheCut(t *testing.T) {
	raw := []byte(strings.Repeat("a", maxLoggedPayload-1) + "é" + "tail")

	got := truncatePayload(raw)

	require.True(t, json.Valid([]byte(`"`+strings.TrimSuffix(got, "...")+`"`)))
	require.Len(t, got, maxLoggedPayload-1+3)
}

func TestReadJSONTruncatesTheDecodedBodyItLogs(t *testing.T) {
	_, lines := captureDebug(t)
	long := strings.Repeat("n", maxLoggedPayload*2)
	req, w := postJSON(`{"name":"` + long + `"}`)

	var dst payload
	require.NoError(t, ReadJSON(w, req, &dst, testMaxBytes))

	entries := lines()
	require.Len(t, entries, 1)
	require.Equal(t, "request decoded", entries[0]["msg"])
	body := entries[0]["body"].(string)
	require.True(t, strings.HasSuffix(body, "..."))
	require.Len(t, body, maxLoggedPayload+3)
}

func TestWriteJSONTruncatesThePayloadItLogs(t *testing.T) {
	_, lines := captureDebug(t)
	w := newRecorder()

	WriteJSON(context.Background(), w, http.StatusOK,
		payload{Name: strings.Repeat("n", maxLoggedPayload*2)})

	entries := lines()
	require.Len(t, entries, 1)
	require.Equal(t, "response sent", entries[0]["msg"])
	require.True(t, strings.HasSuffix(entries[0]["payload"].(string), "..."))
}

// Log Valuer

type secretPayload struct {
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

func (p secretPayload) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("name", p.Name),
		slog.String("secret", "[redacted]"),
	)
}

func TestReadJSONUsesLogValuerOfDecodedBody(t *testing.T) {
	_, lines := captureDebug(t)
	req, w := postJSON(`{"name":"ada","secret":"hunter2"}`)

	var dst secretPayload
	require.NoError(t, ReadJSON(w, req, &dst, testMaxBytes))
	require.Equal(t, "hunter2", dst.Secret)

	entries := lines()
	require.Len(t, entries, 1)
	body := entries[0]["body"].(map[string]any)
	require.Equal(t, "ada", body["name"])
	require.Equal(t, "[redacted]", body["secret"])
}

func TestWriteJSONUsesLogValuerOfResponse(t *testing.T) {
	_, lines := captureDebug(t)
	w := newRecorder()

	WriteJSON(context.Background(), w, http.StatusOK, secretPayload{Name: "ada", Secret: "hunter2"})

	require.Contains(t, w.Body.String(), "hunter2") // the client still gets it

	entries := lines()
	require.Len(t, entries, 1)
	payload := entries[0]["payload"].(map[string]any)
	require.Equal(t, "[redacted]", payload["secret"])
}

type countingValue struct{ n *int }

func (c countingValue) MarshalJSON() ([]byte, error) {
	*c.n++
	return []byte(`"counted"`), nil
}

// the marshal and the cut must not happen when the record is dropped
func TestTruncationIsSkippedWhenDebugIsOff(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	marshalled := 0
	v := countingValue{n: &marshalled}
	slog.Debug("dropped", "body", truncatedValue{v})

	require.Empty(t, buf.String())
	require.Zero(t, marshalled)
}
