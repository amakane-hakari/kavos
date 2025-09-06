package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
)

// AppError はアプリケーション固有のエラーを表します。
type AppError struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Meta    any    `json:"meta,omitempty"`
}

const (
	// CodeBadRequest は 400 Bad Request エラーを表します。
	CodeBadRequest = "BAD_REQUEST"
	// CodeNotFound は 404 Not Found エラーを表します。
	CodeNotFound = "NOT_FOUND"
	// CodeInternalError は 500 Internal Server Error エラーを表します。
	CodeInternalError = "INTERNAL_ERROR"
	// CodeInvalidJSON は 不正なJSONによる 400 Bad Request エラーを表します。
	CodeInvalidJSON = "INVALID_JSON"
	// CodeTimeout は タイムアウトによる 408 Request Timeout エラーを表します。
	CodeTimeout = "TIMEOUT"
	// CodeCanceled は キャンセルによる 408 Request Timeout エラーを表します。
	CodeCanceled = "CANCELED"
	// CodeUnauthorized は 401 Unauthorized エラーを表します。
	CodeUnauthorized = "UNAUTHORIZED"
	// CodeForbidden は 403 Forbidden エラーを表します。
	CodeForbidden = "FORBIDDEN"
	// CodeConflict は 409 Conflict エラーを表します。
	CodeConflict = "CONFLICT"
	// CodeTooManyRequests は 429 Too Many Requests エラーを表します。
	CodeTooManyRequests = "TOO_MANY_REQUESTS"
)

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

// NewAppError は新しい AppError を作成します。
func NewAppError(status int, code, message string, meta any) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Meta:    meta,
	}
}

// BadRequest は 400 Bad Request エラーを表す AppError を作成します。
func BadRequest(msg string) *AppError {
	return NewAppError(http.StatusBadRequest, CodeBadRequest, msg, nil)
}

// NotFound は 404 Not Found エラーを表す AppError を作成します。
func NotFound(msg string) *AppError {
	return NewAppError(http.StatusNotFound, CodeNotFound, msg, nil)
}

// Internal は 500 Internal Server Error エラーを表す AppError を作成します。
func Internal(msg string) *AppError {
	return NewAppError(http.StatusInternalServerError, CodeInternalError, msg, nil)
}

// InvalidJSON は 不正なJSONによる 400 Bad Request エラーを表す AppError を作成します。
func InvalidJSON(msg string) *AppError {
	return NewAppError(http.StatusBadRequest, CodeInvalidJSON, msg, nil)
}

// FromStdError は標準の error を AppError に変換します。
func FromStdError(err error) *AppError {
	if err == nil {
		return nil
	}

	var app *AppError
	if errors.As(err, &app) {
		return app
	}
	switch {
	case errors.Is(err, context.Canceled):
		return NewAppError(http.StatusRequestTimeout, CodeCanceled, "request canceled", nil)
	case errors.Is(err, context.DeadlineExceeded):
		return NewAppError(http.StatusRequestTimeout, CodeTimeout, "request timeout", nil)
	default:
		return Internal("unexpected error")
	}
}

type successEnvelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Err *AppError `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, successEnvelope{Data: data})
}

func writeError(w http.ResponseWriter, err error) {
	var app *AppError
	if errors.As(err, &app) {
		writeJSON(w, app.Status, errorEnvelope{Err: app})
		return
	}
	// Fallback
	writeJSON(w, http.StatusInternalServerError, errorEnvelope{Err: Internal("unexpected error")})
}
