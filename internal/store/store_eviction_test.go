package store

import "testing"

func TestStore_LRUEviction(t *testing.T) {
	s := New[string, string]().WithEvictor(NewLRUEvictor[string, string](2))

	s.Set("a", "1")
	s.Set("b", "2")

	if v, ok := s.Get("a"); !ok || v != "1" {
		t.Fatalf("expected a")
	}

	s.Set("c", "3")

	if _, ok := s.Get("b"); ok {
		t.Fatalf("b should be evicted")
	}
	if _, ok := s.Get("a"); !ok {
		t.Fatalf("a should remain")
	}
	if _, ok := s.Get("c"); !ok {
		t.Fatalf("c should remain")
	}

	s.Set("d", "4")

	if _, ok := s.Get("a"); ok {
		t.Fatalf("a should evicted after adding d")
	}
}