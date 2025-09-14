package store

import "sync"

type shardCompact[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]entry[V]
}

type shardPadding[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]entry[V]
	_  [cacheLineSize]byte // cache line padding
}

func (s *Store[K, V]) getShard(key K) (rw *sync.RWMutex, m map[K]entry[V]) {
	h := s.hashKey(key)
	idx := int(h & s.shardMask)
	if s.cfg.EnableShardPadding {
		sh := &s.shardsPadded[idx]
		return &sh.mu, sh.m
	}
	sh := &s.shardsCompact[idx]
	return &sh.mu, sh.m
}
