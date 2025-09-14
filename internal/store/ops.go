package store

import "time"

// Set はキーと値をストアにセットします。
func (s *Store[K, V]) Set(key K, value V) {
	s.SetWithTTL(key, value, 0)
}

// SetWithTTL はキーと値をストアにセットします。
func (s *Store[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).UnixNano()
	}
	mu, mp := s.getShard(key)
	mu.Lock()
	_, existed := mp[key]
	mp[key] = entry[V]{val: value, expireAt: exp}
	mu.Unlock()

	if existed {
		s.cfg.Metrics.IncSetUpdate()
	} else {
		s.cfg.Metrics.IncSetNew()
	}

	if s.cfg.Logger != nil {
		if existed {
			s.cfg.Logger.Debug("store.update", "key", key)
		} else {
			s.cfg.Logger.Debug("store.set", "key", key, "ttl", ttl.String())
		}
	}

	if s.evictor != nil {
		victims := s.evictor.OnSet(key, value, existed)
		for _, vk := range victims {
			s.deleteInternal(vk, true)
		}
		if len(victims) > 0 {
			s.cfg.Metrics.AddEvicted(len(victims))
			if s.cfg.Logger != nil {
				s.cfg.Logger.Info("store.evict", "count", len(victims), "victims", victims)
			}
		}
		if sp, ok := s.evictor.(interface{ Size() int }); ok {
			s.cfg.Metrics.SetLRUSize(sp.Size())
		}
	}
}

// Get はキーに対応する値を取得します。
func (s *Store[K, V]) Get(key K) (V, bool) {
	mu, mp := s.getShard(key)
	mu.RLock()
	e, exists := mp[key]
	mu.RUnlock()
	if !exists {
		s.cfg.Metrics.IncGetMiss()
		if s.evictor != nil {
			s.evictor.OnGet(key, false)
		}
		var zero V
		return zero, false
	}
	if e.expireAt > 0 && e.expireAt <= time.Now().UnixNano() {
		// 遅延削除
		mu.Lock()
		// 期限内に他ゴルーチンが更新しているか再確認
		cur, still := mp[key]
		if still && cur.expireAt == e.expireAt {
			delete(mp, key)
		}
		mu.Unlock()
		if s.evictor != nil {
			s.evictor.OnDelete(key)
			if sp, ok := s.evictor.(interface{ Size() int }); ok {
				s.cfg.Metrics.SetLRUSize(sp.Size())
			}
		}
		s.cfg.Metrics.IncGetMiss()
		s.cfg.Metrics.AddTTLExpired(1)
		if s.cfg.Logger != nil {
			s.cfg.Logger.Debug("store.ttl.expired", "key", key)
		}
		var zero V
		return zero, false
	}
	s.cfg.Metrics.IncGetHit()
	if s.evictor != nil {
		s.evictor.OnGet(key, true)
	}
	return e.val, true
}

// Delete はキーに対応する値を削除します。
func (s *Store[K, V]) Delete(key K) {
	s.deleteInternal(key, false)
}

func (s *Store[K, V]) deleteInternal(key K, fromEviction bool) {
	mu, mp := s.getShard(key)
	mu.Lock()
	_, existed := mp[key]
	if existed {
		delete(mp, key)
	}
	mu.Unlock()
	if existed && s.evictor != nil && !fromEviction {
		s.evictor.OnDelete(key)
		if sp, ok := s.evictor.(interface{ Size() int }); ok {
			s.cfg.Metrics.SetLRUSize(sp.Size())
		}
	}
}

// Len はストア内のアイテム数を返します。
func (s *Store[K, V]) Len() int {
	now := time.Now().UnixNano()
	total := 0
	if s.cfg.EnableShardPadding {
		for i := range s.shardsPadded {
			sh := &s.shardsPadded[i]
			sh.mu.RLock()
			for _, e := range sh.m {
				if e.expireAt == 0 || e.expireAt > now {
					total++
				}
			}
			sh.mu.RUnlock()
		}
	} else {
		for i := range s.shardsCompact {
			sh := &s.shardsCompact[i]
			sh.mu.RLock()
			for _, e := range sh.m {
				if e.expireAt == 0 || e.expireAt > now {
					total++
				}
			}
			sh.mu.RUnlock()
		}
	}
	return total
}
