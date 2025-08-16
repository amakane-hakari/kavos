package http

import (
	"encoding/json"
	"errors"
	"net/http"
)

type AppError struct {
	Status  int         `json:"status"`
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Meta    interface{} `json:"meta,omitempty"`
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

func NewAppError(status int, code, message string, meta interface{}) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Meta:    meta,
	}
}

func BadRequest(msg string) *AppError {
	return NewAppError(http.StatusBadRequest, "BAD_REQUEST", msg, nil)
}
func NotFound(msg string) *AppError { return NewAppError(http.StatusNotFound, "NOT_FOUND", msg, nil) }
func Internal(msg string) *AppError {
	return NewAppError(http.StatusInternalServerError, "INTERNAL_ERROR", msg, nil)
}

type successEnvelope struct {
	Data interface{} `json:"data"`
}

type errorEnvelope struct {
	Err *AppError `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, status int, data interface{}) {
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
