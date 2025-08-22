package http

import (
	"net/http"

	"github.com/amakane-hakari/kavos/internal/store"
	"github.com/go-chi/chi/v5"

	ilog "github.com/amakane-hakari/kavos/internal/log"
)

func NewRouter(st *store.Store[string, string], logger ilog.Logger) http.Handler {
	r := chi.NewRouter()
	r.Use(requestIDMiddleware, recoverMiddleware)
	r.Use(AccessLog(logger))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	kv := &kvHandler{st: st}
	kv.mount(r)

	return r
}
