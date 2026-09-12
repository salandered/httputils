package httputils

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const testMaxBytes = 1 << 16 // 64 kb

type payload struct {
	Name string `json:"name"`
}

func TestReadJSONDecodesASingleObject(t *testing.T) {
	req, w := postJSON(`{"name":"ada"}`)

	var dst payload
	require.NoError(t, ReadJSON(w, req, &dst, testMaxBytes))
	require.Equal(t, "ada", dst.Name)
}

func TestReadJSONRejectsUnknownFields(t *testing.T) {
	req, w := postJSON(`{"name":"ada","rank":1}`)

	var dst payload
	require.ErrorContains(t, ReadJSON(w, req, &dst, testMaxBytes), "rank")
}

func TestReadJSONRejectsASecondObject(t *testing.T) {
	req, w := postJSON(`{"name":"ada"}{"name":"bob"}`)

	var dst payload
	require.ErrorContains(t, ReadJSON(w, req, &dst, testMaxBytes), "single JSON object")
}

func TestReadJSONRejectsTrailingGarbage(t *testing.T) {
	req, w := postJSON(`{"name":"ada"} not json`)

	var dst payload
	require.ErrorContains(t, ReadJSON(w, req, &dst, testMaxBytes), "single JSON object")
}

func TestReadJSONRejectsABodyOverTheLimit(t *testing.T) {
	big := `{"name":"` + strings.Repeat("x", testMaxBytes) + `"}`
	req, w := postJSON(big)

	var dst payload
	err := ReadJSON(w, req, &dst, testMaxBytes)

	var maxBytesErr *http.MaxBytesError
	require.ErrorAs(t, err, &maxBytesErr)
}

func TestWriteJSONSetsStatusContentTypeAndBody(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(context.Background(), w, http.StatusCreated, payload{Name: "ada"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.JSONEq(t, `{"name":"ada"}`, w.Body.String())
}

func TestWriteJSONFallsBackTo500OnAnUnmarshalableValue(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(context.Background(), w, http.StatusOK, make(chan int))

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
}

func TestReadJSONHonoursTheCapItIsGiven(t *testing.T) {
	req, w := postJSON(`{"name":"abcdefghij"}`)

	var dst payload
	err := ReadJSON(w, req, &dst, 8)

	var maxBytesErr *http.MaxBytesError
	require.ErrorAs(t, err, &maxBytesErr)
}

func TestReadJSONAcceptsABodyThatFitsATightCap(t *testing.T) {
	body := `{"name":"ada"}`
	req, w := postJSON(body)

	var dst payload
	require.NoError(t, ReadJSON(w, req, &dst, int64(len(body))))
	require.Equal(t, "ada", dst.Name)
}

func TestWriteErrorSendsTheRealMessageFor4xx(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(context.Background(), w, errors.New("board closed"), http.StatusConflict)

	require.Equal(t, http.StatusConflict, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	require.JSONEq(t, `{"error":"board closed"}`, w.Body.String())
}

func TestWriteErrorHidesTheCauseFor5xx(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(
		context.Background(), w, errors.New("redis: dial tcp 10.0.0.4:6379"),
		http.StatusInternalServerError)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
	require.NotContains(t, w.Body.String(), "10.0.0.4")
}

func TestWriteErrorHidesTheCauseForAny5xx(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(
		context.Background(), w, errors.New("upstream down"), http.StatusBadGateway)

	require.Equal(t, http.StatusBadGateway, w.Code)
	require.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())
}
