package store

import (
	"sync"
	"time"

	"github.com/amakane-hakari/kavos/internal/metrics"
)

// Store は KVS のストアを表します。
type Store[K comparable, V any] struct {
	cfg             Config
	shardMask       uint32        // Shards が 2^n の場合（hash & mask）で index
	cleanupInterval time.Duration // 0 で無効
	stopCh          chan struct{}
	wg              sync.WaitGroup
	evictor         Evictor[K, V]

	closeOnce sync.Once // Close 多重呼び出し防止

	// どちらか一方だけ使用
	shardsCompact []shardCompact[K, V]
	shardsPadded  []shardPadding[K, V]
}

// New は新しい Store を作成します。
func New[K comparable, V any](opts ...Option) *Store[K, V] {
	cfg := Config{Shards: 16, Metrics: &metrics.Noop{}}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Shards < 1 {
		cfg.Shards = 16
	}
	// 2 の冪に揃える
	cfg.Shards = nextPowerOfTwo(cfg.Shards)

	s := &Store[K, V]{
		cfg:             cfg,
		shardMask:       uint32(cfg.Shards - 1),
		cleanupInterval: cfg.CleanupInterval,
		evictor:         nil,
		stopCh:          make(chan struct{}),
	}
	if cfg.EnableShardPadding {
		s.shardsPadded = make([]shardPadding[K, V], cfg.Shards)
		for i := range s.shardsPadded {
			s.shardsPadded[i].m = make(map[K]entry[V])
		}
	} else {
		s.shardsCompact = make([]shardCompact[K, V], cfg.Shards)
		for i := range s.shardsCompact {
			s.shardsCompact[i].m = make(map[K]entry[V])
		}
	}

	if s.cleanupInterval > 0 {
		s.wg.Add(1)
		go s.cleanupLoop()
	}

	return s
}

// WithEvictor はストアのエビクタを設定するメソッドです。
func (s *Store[K, V]) WithEvictor(ev Evictor[K, V]) *Store[K, V] {
	s.evictor = ev
	return s
}

// Close はストアをクローズします。
func (s *Store[K, V]) Close() {
	s.closeOnce.Do(func() {
		if s.stopCh != nil {
			close(s.stopCh)
		}
	})
	s.wg.Wait()
}
