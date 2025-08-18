package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/amakane-hakari/kavos/internal/store"
	"github.com/go-chi/chi/v5"
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

	var ttlDur time.Duration
	if raw := r.URL.Query().Get("ttl"); raw != "" {
		sec, err := strconv.ParseInt(raw, 10, 64)
		if err == nil && sec > 0 {
			ttlDur = time.Duration(sec) * time.Second
		}
	}

	if ttlDur > 0 {
		h.st.SetWithTTL(key, req.Value, ttlDur)
	} else {
		h.st.Set(key, req.Value)
	}

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
