package http

import "net/http"

// HandlerFunc はエラーハンドリングを行うHTTPハンドラの型です。
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// ServeHTTP はエラーハンドリングを行うHTTPハンドラのServeHTTP実装です。
func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		writeError(w, FromStdError(err))
	}
}
