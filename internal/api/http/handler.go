package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/amakane-hakari/kavos/internal/store"
)

type kvHandler struct {
	st *store.Store
}

func (h *kvHandler) mount(r chi.Router) {
	r.Route("/kvs", func(r chi.Router) {
		r.Put("/{key}", h.put)
		r.Get("/{key}", h.get)
		r.Delete("/{key}", h.del)
	})
}

type valueRequest struct {
	Value string `json:"value"`
}

type valueResponse struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *kvHandler) put(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, valueResponse{Error: "empty key"})
		return
	}
	var req valueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, valueResponse{Error: "invalid json"})
		return
	}
	h.st.Set(key, req.Value)
	writeJSON(w, http.StatusOK, valueResponse{Key: key, Value: req.Value})
}

func (h *kvHandler) get(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, valueResponse{Error: "empty key"})
		return
	}
	v, ok := h.st.Get(key)
	if !ok {
		writeJSON(w, http.StatusNotFound, valueResponse{Error: "not found"})
		return
	}
	writeJSON(w, http.StatusOK, valueResponse{Key: key, Value: v})
}

func (h *kvHandler) del(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		writeJSON(w, http.StatusBadRequest, valueResponse{Error: "empty key"})
		return
	}
	h.st.Delete(key)
	writeJSON(w, http.StatusOK, valueResponse{Key: key})
}