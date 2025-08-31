package store

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStore_SetGetDelete(t *testing.T) {
	s := New[string, string]()

	s.Set("foo", "bar")
	if v, ok := s.Get("foo"); !ok || v != "bar" {
		t.Fatalf("expected bar, got %v", v)
	}

	if _, ok := s.Get("baz"); ok {
		t.Fatalf("expected baz to not exist")
	}

	s.Delete("foo")
	if _, ok := s.Get("foo"); ok {
		t.Fatalf("expected foo to be deleted")
	}
}

func TestStore_Concurrency(t *testing.T) {
	s := New[string, string]()
	const n = 1000
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k := "k" + strconv.Itoa(i)
			s.Set(k, "v")
			if _, ok := s.Get(k); !ok {
				t.Errorf("missing key %s", k)
			}
			s.Delete(k)
		}(i)
	}
	wg.Wait()

	if l := s.Len(); l != 0 {
		t.Fatalf("expected len=0 got %d", l)
	}
}

func TestStore_TTLExpiration(t *testing.T) {
	s := New[string, string]()
	s.SetWithTTL("ephemeral", "x", 50*time.Millisecond)

	if v, ok := s.Get("ephemeral"); !ok || v != "x" {
		t.Fatalf("expected present before expiry")
	}

	time.Sleep(70 * time.Millisecond)

	if _, ok := s.Get("ephemeral"); ok {
		t.Fatalf("expected expired key")
	}
}
