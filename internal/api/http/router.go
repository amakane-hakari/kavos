package http

import (
	"net/http"

	"github.com/amakane-hakari/kavos/internal/store"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	ilog "github.com/amakane-hakari/kavos/internal/log"
)

// NewRouter は KVSのHTTPルーターを作成します。
func NewRouter(st *store.Store[string, string], logger ilog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware, recoverMiddleware)
	r.Use(AccessLog(logger))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Method(http.MethodGet, "/metrics", promhttp.Handler())

	kv := &kvHandler{st: st}
	kv.mount(r)

	return r
}
