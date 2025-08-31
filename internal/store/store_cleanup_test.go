package store

import (
	"testing"
	"time"
)

func TestStore_BackgroundCleanup(t *testing.T) {
	s := New[string, string](WithCleanupInterval(100 * time.Millisecond))
	defer s.Close()

	s.SetWithTTL("k", "v", 30*time.Millisecond)

	if _, ok := s.Get("k"); !ok {
		t.Fatalf("should exist before expiry")
	}

	time.Sleep(70 * time.Millisecond)

	if _, ok := s.Get("k"); ok {
		t.Fatalf("expected cleaned key")
	}
}
