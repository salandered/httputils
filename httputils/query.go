package httputils

import (
	"fmt"
	"net/http"
	"strconv"
)

// max <= 0 means no cap
func ParseIntQuery(req *http.Request, name string, def, min_, max_ int64) (int64, error) {
	raw := req.URL.Query().Get(name)
	if raw == "" {
		return def, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid query param, want an integer; param '%v', value '%v'", name, raw)
	}
	if v < min_ || (v > max_ && max_ > 0) {
		if max_ > 0 {
			return 0, fmt.Errorf(
				"invalid query param, want an integer in [%v, %v]; param '%v', value '%v'",
				min_, max_, name, raw,
			)
		}
		return 0, fmt.Errorf(
			"invalid query param, want an integer >= %v; param '%v', value '%v'", min_, name, raw)
	}
	return v, nil
}
