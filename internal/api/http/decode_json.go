package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// DecodeJSON はリクエストボディのJSONをデコードします。
func DecodeJSON(r *http.Request, dst any) error {
	if r.Body == nil {
		return InvalidJSON("empty body")
	}
	defer func() {
		_ = r.Body.Close()
	}()

	const maxSize = 1 << 20 // 1MB
	lim := io.LimitReader(r.Body, maxSize)

	dec := json.NewDecoder(lim)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var se *json.SyntaxError
		var ute *json.UnmarshalTypeError
		switch {
		case errors.As(err, &se):
			return InvalidJSON("malformed JSON")
		case errors.As(err, &ute):
			return InvalidJSON("type mismatch in JSON")
		default:
			return InvalidJSON("invalid JSON")
		}
	}
	// 余分なトークンがないか確認(多重JSON防止)
	if dec.More() {
		return InvalidJSON("multiple JSON values")
	}
	return nil
}
