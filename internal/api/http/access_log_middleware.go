package http

import (
	"net/http"
	"time"

	ilog "github.com/amakane-hakari/kavos/internal/log"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

// AccessLog はリクエストのアクセスログを記録するミドルウェアです。
func AccessLog(l ilog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l == nil {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			lrw := &loggingResponseWriter{ResponseWriter: w}

			next.ServeHTTP(lrw, r)

			dur := time.Since(start)
			l.Info("access.log",
				"method", r.Method,
				"path", r.URL.Path,
				"status", lrw.status,
				"duration_ms", dur.Milliseconds(),
				"bytes", lrw.size,
				"remote", remoteIP(r),
			)
		})
	}
}

func remoteIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		return ip
	}
	return r.RemoteAddr
}
