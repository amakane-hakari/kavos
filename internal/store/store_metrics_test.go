package store

import (
	"testing"
	"time"

	"github.com/amakane-hakari/kavos/internal/metrics"
)

type testMetrics struct {
	m *metrics.Simple
}

func newTestMetrics() (*testMetrics, *metrics.Simple) {
	s := metrics.NewSimple()
	return &testMetrics{m: s}, s
}

func TestStore_MetricsBasic(t *testing.T) {
	tm, simple := newTestMetrics()
	s := New[string, string](WithMetrics(tm.m))
	s.Set("a", "1")
	s.Set("a", "2")
	s.SetWithTTL("b", "3", 30*time.Millisecond)
	_, _ = s.Get("a")
	_, _ = s.Get("missing")
	time.Sleep(40 * time.Millisecond)
	_, _ = s.Get("b")

	if simple.SetNew.Load() != 2 {
		t.Fatalf("SetNew want 2 got %d", simple.SetNew.Load())
	}
	if simple.SetUpdate.Load() != 1 {
		t.Fatalf("SetUpdate want 1 got %d", simple.SetUpdate.Load())
	}
	if simple.GetHit.Load() != 1 {
		t.Fatalf("GetHit want 1 got %d", simple.GetHit.Load())
	}
	if simple.GetMiss.Load() != 2 {
		t.Fatalf("GetMiss want 2 got %d", simple.GetMiss.Load())
	}
	if simple.TTLExpired.Load() != 1 {
		t.Fatalf("TTLExpired want 1 got %d", simple.TTLExpired.Load())
	}
}
