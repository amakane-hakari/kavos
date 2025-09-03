package store

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/amakane-hakari/kavos/internal/metrics"
)

type benchConfig struct {
	shards    int
	readRatio float64
	withEvict bool
	capacity  int
	warmKeys  int
	parallel  bool
}

var benchMatrix = []benchConfig{
	{shards: 1, readRatio: 0.90, withEvict: false, warmKeys: 50_000, parallel: true},
	{shards: 16, readRatio: 0.90, withEvict: false, warmKeys: 50_000, parallel: true},
	{shards: 64, readRatio: 0.90, withEvict: false, warmKeys: 50_000, parallel: true},
	{shards: 256, readRatio: 0.90, withEvict: false, warmKeys: 50_000, parallel: true},

	{shards: 16, readRatio: 0.50, withEvict: false, warmKeys: 50_000, parallel: true},
	{shards: 16, readRatio: 0.10, withEvict: false, warmKeys: 50_000, parallel: true},

	// Eviction enabled
	{shards: 16, readRatio: 0.90, withEvict: true, capacity: 30_000, warmKeys: 50_000, parallel: true},
	{shards: 16, readRatio: 0.50, withEvict: true, capacity: 30_000, warmKeys: 50_000, parallel: true},
	{shards: 16, readRatio: 0.10, withEvict: true, capacity: 30_000, warmKeys: 50_000, parallel: true},

	// serial
	{shards: 16, readRatio: 0.90, withEvict: false, warmKeys: 50_000, parallel: false},
}

func BenchmarkStore_MixedWorkload(b *testing.B) {
	runtime.GC()

	for _, cfg := range benchMatrix {
		name := fmt.Sprintf("shards=%d, readRatio=%.0f, withEvict=%t, warmKeys=%d, parallel=%t",
			cfg.shards, cfg.readRatio*100, cfg.withEvict, cfg.warmKeys, cfg.parallel,
		)
		b.Run(name, func(b *testing.B) {
			runOneBenchmark(b, cfg)
		})
	}
}

func runOneBenchmark(b *testing.B, cfg benchConfig) {
	b.ReportAllocs()

	// 乱数(固定シードで再現性確保)
	rnd := rand.New(rand.NewSource(42))

	mx := metrics.Noop{}
	st := New[string, string](
		WithShards(cfg.shards),
		WithMetrics(&mx),
	)
	if cfg.withEvict {
		st.WithEvictor(NewLRUEvictor[string, string](cfg.capacity))
	}

	// ウォームアップ
	keys := make([]string, cfg.warmKeys)
	for i := 0; i < cfg.warmKeys; i++ {
		k := fmt.Sprintf("k%05d", i)
		v := fmt.Sprintf("v%05d", i)
		st.Set(k, v)
		keys[i] = k
	}

	var setCounter atomic.Uint64
	var getHit atomic.Int64

	work := func(iters int, r *rand.Rand) {
		localLen := len(keys)
		for i := 0; i < iters; i++ {
			// Get or Set 判定
			if r.Float64() < cfg.readRatio {
				k := keys[r.Intn(localLen)]
				if _, ok := st.Get(k); ok {
					getHit.Add(1)
				}
			} else {
				// 新規 or 既存更新を混合 (10% 新規: eviction 誘発)
				if r.Intn(10) == 0 {
					k := fmt.Sprintf("n%d_%d", r.Intn(1_000_000), i)
					st.Set(k, "x")
				} else {
					k := keys[r.Intn(localLen)]
					st.Set(k, "u")
				}
				setCounter.Add(1)
			}
		}
	}

	if cfg.parallel {
		b.SetParallelism(runtime.GOMAXPROCS(0)) // 1:1 目安
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			// 各ゴルーチン個別 rand
			rLocal := rand.New(rand.NewSource(rnd.Int63()))
			for pb.Next() {
				work(1, rLocal)
			}
		})
	} else {
		b.ResetTimer()
		rLocal := rand.New(rand.NewSource(rnd.Int63()))
		for i := 0; i < b.N; i++ {
			work(1, rLocal)
		}
	}

	b.StopTimer()

	b.ReportMetric(float64(setCounter.Load()), "sets_total")
	b.ReportMetric(float64(getHit.Load()), "get_hits_total")
}

func BenchmarkLRUOnly(b *testing.B) {
	ev := NewLRUEvictor[string, struct{}](100_000)
	r := rand.New(rand.NewSource(42))

	// プリウォーム
	for i := 0; i < 90_000; i++ {
		ev.OnSet(fmt.Sprintf("k%d", i), struct{}{}, false)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		k := fmt.Sprintf("k%d", r.Intn(150_000))
		victims := ev.OnSet(k, struct{}{}, true)
		_ = victims
		ev.OnGet(k, true)
	}
}
