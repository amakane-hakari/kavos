package http

import (
	"net/http"
	"sync/atomic"
)

var draining atomic.Bool

// SetDraining はドレイニング状態を設定します。
func SetDraining(v bool) {
	draining.Store(v)
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	if draining.Load() {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"draining"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
