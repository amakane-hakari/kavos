package store

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"
)

type entry[V any] struct {
	val      V
	expireAt int64 // 0 = no expiry (UnixNano)
}

type shard[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]entry[V]
}

type Config struct {
	Shards int // 2 の冪推奨。0/未指定なら 16
}

type Option func(*Config)

func WithShards(n int) Option {
	return func(c *Config) { c.Shards = n }
}

type Store[K comparable, V any] struct {
	cfg       Config
	shards    []shard[K, V]
	shardMask uint32 // Shards が 2^n の場合（hash & mask）で index
}

func New[K comparable, V any](opts ...Option) *Store[K, V] {
	cfg := Config{Shards: 16}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Shards < 1 {
		cfg.Shards = 16
	}
	// 2 の冪に揃える
	cfg.Shards = nextPowerOfTwo(cfg.Shards)

	s := &Store[K, V]{
		cfg:       cfg,
		shards:    make([]shard[K, V], cfg.Shards),
		shardMask: uint32(cfg.Shards - 1),
	}
	for i := range s.shards {
		s.shards[i].m = make(map[K]entry[V])
	}

	return s
}

func (s *Store[K, V]) Set(key K, value V) {
	s.SetWithTTL(key, value, 0)
}

func (s *Store[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	sh := s.getShard(key)
	sh.mu.Lock()
	sh.m[key] = entry[V]{val: value, expireAt: exp}
	sh.mu.Unlock()
}

func (s *Store[K, V]) Get(key K) (V, bool) {
	sh := s.getShard(key)
	sh.mu.RLock()
	e, exists := sh.m[key]
	sh.mu.RUnlock()
	if !exists {
		var zero V
		return zero, false
	}
	if e.expireAt > 0 && e.expireAt <= time.Now().UnixNano() {
		// 遅延削除
		sh.mu.Lock()
		// 期限内に他ゴルーチンが更新しているか再確認
		cur, still := sh.m[key]
		if still && cur.expireAt == e.expireAt {
			delete(sh.m, key)
		}
		sh.mu.Unlock()
		var zero V
		return zero, false
	}
	return e.val, true
}

func (s *Store[K, V]) Delete(key K) {
	sh := s.getShard(key)
	sh.mu.Lock()
	delete(sh.m, key)
	sh.mu.Unlock()
}

func (s *Store[K, V]) Len() int {
	now := time.Now().UnixNano()
	total := 0
	for i := range s.shards {
		sh := &s.shards[i]
		sh.mu.RLock()
		for _, e := range sh.m {
			if e.expireAt == 0 || e.expireAt > now {
				total++
			}
		}
		sh.mu.RUnlock()
	}
	return total
}

func (s *Store[K, V]) getShard(key K) *shard[K, V] {
	h := s.hashKey(key)
	idx := int(h & s.shardMask)
	return &s.shards[idx]
}

func (s *Store[K, V]) hashKey(key K) uint32 {
	switch k := any(key).(type) {
	case string:
		h := fnv.New32a()
		_, _ = h.Write([]byte(k))
		return h.Sum32()
	case int:
		return uint32(k)
	case int32:
		return uint32(k)
	case int64:
		return uint32(k) ^ uint32(k>>32)
	case uint:
		return uint32(k)
	case uint32:
		return k
	case uint64:
		return uint32(k) ^ uint32(k>>32)
	default:
		h := fnv.New32a()
		_, _ = h.Write([]byte(fmt.Sprintf("%v", k)))
		return h.Sum32()
	}
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}
