package store

import "time"

func (s *Store[K, V]) cleanupLoop() {
	defer s.wg.Done()
	t := time.NewTicker(s.cleanupInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.scanExpired()
		case <-s.stopCh:
			return
		}
	}
}

func (s *Store[K, V]) scanExpired() {
	now := time.Now().UnixNano()
	totalExpired := 0
	if s.cfg.EnableShardPadding {
		for i := range s.shardsPadded {
			sh := &s.shardsPadded[i]
			var expiredKeys []K
			sh.mu.Lock()
			for k, e := range sh.m {
				if e.expireAt > 0 && e.expireAt <= now {
					delete(sh.m, k)
					expiredKeys = append(expiredKeys, k)
				}
			}
			sh.mu.Unlock()
			if len(expiredKeys) > 0 {
				totalExpired += len(expiredKeys)
				if s.evictor != nil {
					for _, k := range expiredKeys {
						s.evictor.OnDelete(k)
					}
				}
				if sp, ok := s.evictor.(interface{ Size() int }); ok {
					s.cfg.Metrics.SetLRUSize(sp.Size())
				}
				if s.cfg.Logger != nil {
					s.cfg.Logger.Info("store.ttl.cleanup", "shard", i, "removed", len(expiredKeys))
				}
			}
		}
	} else {
		for i := range s.shardsCompact {
			sh := &s.shardsCompact[i]
			var expiredKeys []K
			sh.mu.Lock()
			for k, e := range sh.m {
				if e.expireAt > 0 && e.expireAt <= now {
					delete(sh.m, k)
					expiredKeys = append(expiredKeys, k)
				}
			}
			if s.cfg.Logger != nil && len(expiredKeys) > 0 {
				s.cfg.Logger.Info("store.ttl.cleanup", "shard", i, "removed", len(expiredKeys))
			}
			sh.mu.Unlock()
			if len(expiredKeys) > 0 {
				totalExpired += len(expiredKeys)
				if s.evictor != nil {
					for _, k := range expiredKeys {
						s.evictor.OnDelete(k)
					}
				}
				if sp, ok := s.evictor.(interface{ Size() int }); ok {
					s.cfg.Metrics.SetLRUSize(sp.Size())
				}
				if s.cfg.Logger != nil {
					s.cfg.Logger.Info("store.ttl.cleanup", "shard", i, "removed", len(expiredKeys))
				}
			}
		}
	}
	if totalExpired > 0 {
		s.cfg.Metrics.AddTTLExpired(totalExpired)
	}
}
