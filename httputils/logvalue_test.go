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

func TestTruncatePayloadLeavesAShortPayloadAlone(t *testing.T) {
	require.Equal(t, `{"a":1}`, truncatePayload([]byte(`{"a":1}`)))
}

func TestTruncatePayloadCutsAndMarksALongPayload(t *testing.T) {
	got := truncatePayload([]byte(strings.Repeat("a", maxLoggedPayload+10)))

	require.True(t, strings.HasSuffix(got, "..."))
	require.Len(t, got, maxLoggedPayload+3) // 3 - for three dots
}

func TestTruncatePayloadDropsARuneSplitByTheCut(t *testing.T) {
	raw := []byte(strings.Repeat("a", maxLoggedPayload-1) + "é" + "tail")

	got := truncatePayload(raw)

	require.True(t, json.Valid([]byte(`"`+strings.TrimSuffix(got, "...")+`"`)))
	require.Len(t, got, maxLoggedPayload-1+3)
}

// captures records at Debug and hands back the decoded lines
func captureDebug(t *testing.T) (*bytes.Buffer, func() []map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return &buf, func() []map[string]any {
		var out []map[string]any
		for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
			if line == "" {
				continue
			}
			var entry map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &entry))
			out = append(out, entry)
		}
		return out
	}
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

type countingValue struct{ n *int }

func (c countingValue) MarshalJSON() ([]byte, error) {
	*c.n++
	return []byte(`"counted"`), nil
}
