package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Prom は Prometheus を使ったメトリクス実装です。
type Prom struct {
	setNew     prometheus.Counter
	setUpdate  prometheus.Counter
	getHit     prometheus.Counter
	getMiss    prometheus.Counter
	evicted    prometheus.Counter
	ttlExpired prometheus.Counter
	lruSize    prometheus.Gauge
}

// NewProm は Prometheus を使ったメトリクス実装を初期化します。
func NewProm(namespace string) *Prom {
	makeC := func(name, help string) prometheus.Counter {
		return prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		})
	}
	makeG := func(name, help string) prometheus.Gauge {
		return prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      name,
			Help:      help,
		})
	}

	p := &Prom{
		setNew:     makeC("set_new_total", "Number of new keys set"),
		setUpdate:  makeC("set_update_total", "Number of keys updated"),
		getHit:     makeC("get_hit_total", "Number of cache hits"),
		getMiss:    makeC("get_miss_total", "Number of cache misses"),
		evicted:    makeC("evicted_total", "Number of evicted items"),
		ttlExpired: makeC("ttl_expired_total", "Number of TTL expired items"),
		lruSize:    makeG("lru_current_size", "Current number of keys tracked by LRU"),
	}

	// Register (重複登録は無視したいので MustRegister で panic するなら再利用側で 1 回だけ呼ぶ設計)
	prometheus.MustRegister(
		p.setNew, p.setUpdate, p.getHit, p.getMiss, p.evicted, p.ttlExpired, p.lruSize,
	)
	return p
}

// IncSetNew は新しいキーが追加されたことをカウントします。
func (p *Prom) IncSetNew() { p.setNew.Inc() }

// IncSetUpdate は既存のキーが更新されたことをカウントします。
func (p *Prom) IncSetUpdate() { p.setUpdate.Inc() }

// IncGetHit はキャッシュヒットをカウントします。
func (p *Prom) IncGetHit() { p.getHit.Inc() }

// IncGetMiss はキャッシュミスをカウントします。
func (p *Prom) IncGetMiss() { p.getMiss.Inc() }

// AddEvicted は追い出されたアイテムの数を加算します。
func (p *Prom) AddEvicted(n int) {
	if n > 0 {
		p.evicted.Add(float64(n))
	}
}

// AddTTLExpired はTTLが期限切れになったアイテムの数を加算します。
func (p *Prom) AddTTLExpired(n int) {
	if n > 0 {
		p.ttlExpired.Add(float64(n))
	}
}

// SetLRUSize は LRU サイズを設定します。
func (p *Prom) SetLRUSize(n int) {
	if n >= 0 {
		p.lruSize.Set(float64(n))
	}
}
