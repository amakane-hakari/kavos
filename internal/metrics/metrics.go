package metrics

import (
	"sync/atomic"
)

// Interface はメトリクス更新用抽象
type Interface interface {
	IncSetNew()
	IncSetUpdate()
	IncGetHit()
	IncGetMiss()
	AddEvicted(n int)
	AddTTLExpired(n int)
	SetLRUSize(n int)
}

// Noop は何もしないメトリクス実装
type Noop struct{}

// IncSetNew は何もしないメトリクス実装
func (Noop) IncSetNew() {}

// IncSetUpdate は何もしないメトリクス実装
func (Noop) IncSetUpdate() {}

// IncGetHit は何もしないメトリクス実装
func (Noop) IncGetHit() {}

// IncGetMiss は何もしないメトリクス実装
func (Noop) IncGetMiss() {}

// AddEvicted は何もしないメトリクス実装
func (Noop) AddEvicted(_ int) {}

// AddTTLExpired は何もしないメトリクス実装
func (Noop) AddTTLExpired(_ int) {}

// SetLRUSize は何もしないメトリクス実装
func (Noop) SetLRUSize(_ int) {}

// Simple はシンプルなメトリクス実装です。
type Simple struct {
	SetNew     atomic.Uint64
	SetUpdate  atomic.Uint64
	GetHit     atomic.Uint64
	GetMiss    atomic.Uint64
	Evicted    atomic.Uint64
	TTLExpired atomic.Uint64
	LRUSize    atomic.Uint64
}

// NewSimple は新しい Simple メトリクスを作成します。
func NewSimple() *Simple { return &Simple{} }

// IncSetNew は新しいキーが追加されたことをカウントします。
func (m *Simple) IncSetNew() { m.SetNew.Add(1) }

// IncSetUpdate は既存のキーが更新されたことをカウントします。
func (m *Simple) IncSetUpdate() { m.SetUpdate.Add(1) }

// IncGetHit はキャッシュヒットをカウントします。
func (m *Simple) IncGetHit() { m.GetHit.Add(1) }

// IncGetMiss はキャッシュミスをカウントします。
func (m *Simple) IncGetMiss() { m.GetMiss.Add(1) }

// AddEvicted はエビクションされたアイテムの数を加算します。
func (m *Simple) AddEvicted(n int) {
	if n > 0 {
		m.Evicted.Add(uint64(n))
	}
}

// AddTTLExpired は TTL が期限切れになったアイテムの数を加算します。
func (m *Simple) AddTTLExpired(n int) {
	if n > 0 {
		m.TTLExpired.Add(uint64(n))
	}
}

// SetLRUSize は LRU サイズを設定します。
func (m *Simple) SetLRUSize(n int) {
	if n >= 0 {
		m.LRUSize.Store(uint64(n))
	}
}
