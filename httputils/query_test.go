package httputils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIntQuery(t *testing.T) {
	tests := []struct {
		raw       string // omitted from the URL when empty
		def       int64
		min_      int64
		max_      int64 // <= 0 means no cap
		wantValue int64
		wantErr   bool
		name      string
	}{
		{raw: "", def: 10, min_: 0, max_: 100, wantValue: 10, name: "missing param returns default"},
		{raw: "5", def: 10, min_: 0, max_: 100, wantValue: 5, name: "value within range"},
		{raw: "0", def: 10, min_: 0, max_: 100, wantValue: 0, name: "value at min boundary"},
		{raw: "100", def: 10, min_: 0, max_: 100, wantValue: 100, name: "value at max boundary"},
		{raw: "1000000", def: 10, min_: 0, max_: 0, wantValue: 1000000, name: "no cap allows large value"},
		{raw: "abc", def: 10, min_: 0, max_: 100, wantErr: true, name: "not an integer"},
		{raw: "-1", def: 10, min_: 0, max_: 100, wantErr: true, name: "below min"},
		{raw: "101", def: 10, min_: 0, max_: 100, wantErr: true, name: "above max"},
		{raw: "-1", def: 10, min_: 0, max_: 0, wantErr: true, name: "below min with no cap"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.raw != "" {
				url = fmt.Sprintf("/?%s=%s", "param", tt.raw)
			}
			req := httptest.NewRequest(http.MethodGet, url, nil)

			v, err := ParseIntQuery(req, "param", tt.def, tt.min_, tt.max_)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantValue, v)
		})
	}
}

func TestParseIntQueryNamesTheParamAndValueInEveryError(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?limit=999", nil)

	_, err := ParseIntQuery(req, "limit", 10, 1, 100)

	require.ErrorContains(t, err, "limit")
	require.ErrorContains(t, err, "999")
	require.ErrorContains(t, err, "[1, 100]")
}

func TestParseIntQueryReportsOnlyTheLowerBoundWhenUncapped(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?offset=-5", nil)

	_, err := ParseIntQuery(req, "offset", 0, 0, 0)

	require.ErrorContains(t, err, ">= 0")
	require.ErrorContains(t, err, "offset")
	require.ErrorContains(t, err, "-5")
}
