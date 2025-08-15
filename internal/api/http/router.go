package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/amakane-hakari/kavos/internal/store"
)

func NewRouter(st *store.Store) http.Handler {
	r := chi.NewRouter()
	r.Get("/health", healthHandler)

	kv := &kvHandler{st: st}
	kv.mount(r)

	return r
}

type healthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{Status: "ok"}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
