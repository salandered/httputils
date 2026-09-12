package httputils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Decodes the incoming req.
// Checks: only one JSON object; bounded by maxBytes; no unknown fields.
// maxBytes must be positive; an oversized body fails with [http.MaxBytesError].
func ReadJSON(w http.ResponseWriter, req *http.Request, dst any, maxBytes int64) error {
	req.Body = http.MaxBytesReader(w, req.Body, maxBytes)
	dec := json.NewDecoder(req.Body)
	dec.DisallowUnknownFields()

	// Consider adding branches for json errors like json.SyntaxError, json.UnmarshalTypeError, etc
	if err := dec.Decode(dst); err != nil {
		return err
	}

	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("body must contain a single JSON object")
	}

	slog.DebugContext(req.Context(), "request decoded", "body", truncatedValue{dst})
	return nil
}

func WriteJSON(ctx context.Context, w http.ResponseWriter, statusCode int, data any) {
	rawJSON, err := json.Marshal(data)
	if err != nil {
		WriteError(
			ctx,
			w,
			fmt.Errorf("marshalling response body: %w", err),
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode) // before Write

	_, err = w.Write(rawJSON)
	if err != nil {
		// headers with the status code were already sent to the client
		slog.ErrorContext(ctx, "failed writing response body", "status", statusCode, "error", err)
		return
	}

	// not using rawJSON: marshalled bytes lose their LogValuer (if any).
	// For example, this prevents a response type from redacting itself.
	slog.DebugContext(ctx, "response sent",
		"bytes", len(rawJSON), "payload", truncatedValue{data})
}

type errorResponse struct {
	Error string `json:"error"`
}

func WriteError(ctx context.Context, w http.ResponseWriter, err error, statusCode int) {
	msg := err.Error()
	if statusCode >= http.StatusInternalServerError {
		slog.ErrorContext(ctx, "request failed", "status", statusCode, "error", err)
		msg = "internal server error" // the client should not see the actual error
	} else {
		slog.WarnContext(ctx, "request rejected", "status", statusCode, "error", err)
	}

	rawJSON, marshalErr := json.Marshal(errorResponse{Error: msg})
	if marshalErr != nil {
		slog.ErrorContext(ctx, "failed marshalling error response", "error", marshalErr)
		// all bad, just return a plain text
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	// w.Header().Set("X-Content-Type-Options", "nosniff") // consider adding
	w.WriteHeader(statusCode)

	if _, writeErr := w.Write(rawJSON); writeErr != nil {
		slog.ErrorContext(ctx, "failed writing error response", "status", statusCode, "error", writeErr)
	}
}
