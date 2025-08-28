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

// Config はストアの設定を表します。
type Config struct {
	Shards          int           // 2 の冪推奨。0/未指定なら 16
	CleanupInterval time.Duration // 0 で無効
	Logger          logLike
}

type logLike interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Evictor はストアのエビクタインターフェースを表します。
type Evictor[K comparable, V any] interface {
	// keyをセットした（existed: 既存だったか）後に呼ぶ。
	// 返却 victims は Evictor 内部状態から既に除外済みで、Store 側が map から削除する。
	OnSet(key K, value V, existed bool) (victims []K)
	// Get 成功/失敗で呼ぶ（hit=true ならヒット）
	OnGet(key K, hit bool)
	// 明示削除/TTL 遅延削除時（eviction 起因以外）
	OnDelete(key K)
}

// Option はストアのオプションを設定する関数です。
type Option func(*Config)

// WithLogger はストアのロガーを設定するオプションです。
func WithLogger(l logLike) Option {
	return func(c *Config) { c.Logger = l }
}

// WithShards はストアのシャード数を設定するオプションです。
func WithShards(n int) Option {
	return func(c *Config) { c.Shards = n }
}

// WithCleanupInterval はストアのクリーンアップ間隔を設定するオプションです。
func WithCleanupInterval(d time.Duration) Option {
	return func(c *Config) { c.CleanupInterval = d }
}

// Store は KVS のストアを表します。
type Store[K comparable, V any] struct {
	cfg             Config
	shards          []shard[K, V]
	shardMask       uint32 // Shards が 2^n の場合（hash & mask）で index
	cleanupInterval time.Duration // 0 で無効
	stopCh          chan struct{}
	wg              sync.WaitGroup

	evictor Evictor[K, V]
}

// New は新しい Store を作成します。
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
		cfg:             cfg,
		shards:          make([]shard[K, V], cfg.Shards),
		shardMask:       uint32(cfg.Shards - 1),
		cleanupInterval: cfg.CleanupInterval,
		evictor:         nil,
	}
	for i := range s.shards {
		s.shards[i].m = make(map[K]entry[V])
	}

	if s.cleanupInterval > 0 {
		s.stopCh = make(chan struct{})
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
	sh := s.getShard(key)
	sh.mu.Lock()
	_, existed := sh.m[key]
	sh.m[key] = entry[V]{val: value, expireAt: exp}
	sh.mu.Unlock()

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
		if s.cfg.Logger != nil && len(victims) > 0 {
			s.cfg.Logger.Info("store.evict", "count", len(victims), "victims", victims)
		}
	}
}

// Get はキーに対応する値を取得します。
func (s *Store[K, V]) Get(key K) (V, bool) {
	sh := s.getShard(key)
	sh.mu.RLock()
	e, exists := sh.m[key]
	sh.mu.RUnlock()
	if !exists {
		if s.evictor != nil {
			s.evictor.OnGet(key, false)
		}
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
		if s.evictor != nil {
			s.evictor.OnDelete(key)
		}
		var zero V
		return zero, false
	}
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
	sh := s.getShard(key)
	sh.mu.Lock()
	_, existed := sh.m[key]
	if existed {
		delete(sh.m, key)
	}
	sh.mu.Unlock()
	if existed && s.evictor != nil && !fromEviction {
		s.evictor.OnDelete(key)
	}
}

// Len はストア内のアイテム数を返します。
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

// Close はストアをクローズします。
func (s *Store[K, V]) Close() {
	if s.stopCh == nil {
		return
	}
	close(s.stopCh)
	s.wg.Wait()
}

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
	for i := range s.shards {
		sh := &s.shards[i]
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
		if s.evictor != nil {
			for _, k := range expiredKeys {
				s.evictor.OnDelete(k)
			}
		}
	}
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
		_, _ = fmt.Fprintf(h, "%v", k)
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
