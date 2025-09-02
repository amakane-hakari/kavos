package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"runtime/debug"
)

type ctxKey int

const requestIDKey ctxKey = iota

const headerRequestID = "X-Request-ID"

// RequestIDFromContext はコンテキストからリクエストIDを取得します。
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RequestIDMiddleware はリクエストIDを管理するミドルウェアです。
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get(headerRequestID)
			if rid == "" {
				rid = genRequestID()
			}
			w.Header().Set(headerRequestID, rid)
			ctx := context.WithValue(r.Context(), requestIDKey, rid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetRequestID はコンテキストからリクエストIDを取得します。
func GetRequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// RecoverMiddleware はリカバリを行うミドルウェアです。
func RecoverMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					_ = rec
					_ = debug.Stack()
					writeError(w, Internal("panic recovered"))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func genRequestID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
