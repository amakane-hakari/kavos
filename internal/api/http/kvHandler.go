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
		r.Put("/{key}", wrap(h.put))
		r.Get("/{key}", wrap(h.get))
		r.Delete("/{key}", wrap(h.del))
	})
}

type valueRequest struct {
	Value string `json:"value"`
}

type valueDTO struct {
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

func wrap(h handlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			writeError(w, err)
		}
	}
}

func (h *kvHandler) put(w http.ResponseWriter, r *http.Request) error {
	key := chi.URLParam(r, "key")
	if key == "" {
		return BadRequest("empty key")
	}
	var req valueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return BadRequest("invalid json")
	}
	h.st.Set(key, req.Value)
	writeSuccess(w, http.StatusOK, valueDTO{Key: key, Value: req.Value})
	return nil
}

func (h *kvHandler) get(w http.ResponseWriter, r *http.Request) error {
	key := chi.URLParam(r, "key")
	if key == "" {
		return BadRequest("empty key")
	}
	v, ok := h.st.Get(key)
	if !ok {
		return NotFound("key not found")
	}
	writeSuccess(w, http.StatusOK, valueDTO{Key: key, Value: v})
	return nil
}

func (h *kvHandler) del(w http.ResponseWriter, r *http.Request) error {
	key := chi.URLParam(r, "key")
	if key == "" {
		return BadRequest("empty key")
	}
	h.st.Delete(key)
	writeSuccess(w, http.StatusOK, valueDTO{Key: key})
	return nil
}