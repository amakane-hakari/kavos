package store

import (
	"sync"
	"time"
)

type entry struct {
	val      string
	expireAt int64 // 0 = no expiry (UnixNano)
}

type Store struct {
	mu   sync.RWMutex
	data map[string]entry
}

func New() *Store {
	return &Store{
		data: make(map[string]entry),
	}
}

func (s *Store) Set(key, value string) {
	s.SetWithTTL(key, value, 0)
}

func (s *Store) SetWithTTL(key, value string, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	s.mu.Lock()
	s.data[key] = entry{val: value, expireAt: exp}
	s.mu.Unlock()
}

func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	e, ok := s.data[key]
	s.mu.RUnlock()
	if !ok {
		return "", false
	}
	if e.expireAt > 0 && e.expireAt <= time.Now().UnixNano() {
		// 遅延削除
		s.mu.Lock()
		// 期限内に他ゴルーチンが更新しているか再確認
		cur, still := s.data[key]
		if still && cur.expireAt == e.expireAt {
			delete(s.data, key)
		}
		s.mu.Unlock()
		return "", false
	}
	return e.val, true
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()
}

func (s *Store) Len() int {
	now := time.Now().UnixNano()
	s.mu.RLock()
	n := 0
	for _, e := range s.data {
		if e.expireAt == 0 || e.expireAt > now {
			n++
		}
	}
	s.mu.RUnlock()
	return n
}
