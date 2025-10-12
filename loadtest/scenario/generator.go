package scenario

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

// Generator は 負荷試験のターゲットを生成する構造体です。
type Generator struct {
	BaseURL   string
	Keys      int
	ReadRatio float64
	ValueSize int
	TTLRatio  float64
	TTLms     int
	ReadOnly  bool

	rnd *rand.Rand
	mu  sync.Mutex
	buf []byte
}

// NewGenerator は 指定されたパラメータに基づいて新しい Generator を作成します。
func NewGenerator(base string, keys int, readRatio float64, valueSize int, ttlRatio float64, ttlms int, readOnly bool) *Generator {
	src := rand.NewSource(time.Now().UnixNano())
	return &Generator{
		BaseURL:   base,
		Keys:      keys,
		ReadRatio: clamp(readRatio, 0, 1),
		ValueSize: valueSize,
		TTLRatio:  clamp(ttlRatio, 0, 1),
		TTLms:     ttlms,
		ReadOnly:  readOnly,
		rnd:       rand.New(src),
		buf:       make([]byte, valueSize),
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Targeter は vegeta.Targeter インターフェースを実装し、負荷試験のターゲットを生成します。
func (g *Generator) Targeter() vegeta.Targeter {
	return func(t *vegeta.Target) error {
		g.mu.Lock()
		defer g.mu.Unlock()

		k := g.rnd.Intn(g.Keys)
		key := fmt.Sprintf("k%06d", k)

		isGet := g.ReadOnly
		if !g.ReadOnly {
			if g.rnd.Float64() < g.ReadRatio {
				isGet = true
			}
		}

		if isGet {
			t.Method = "GET"
			t.URL = fmt.Sprintf("%s/kvs/%s", g.BaseURL, key)
			t.Body = nil
			t.Header = nil
			return nil
		}

		fillRandomLetters(g.rnd, g.buf)
		bodyObj := map[string]any{
			"value": string(g.buf),
		}
		if g.TTLms > 0 && g.rnd.Float64() < g.TTLRatio {
			bodyObj["ttl_ms"] = g.TTLms
		}
		b, err := json.Marshal(bodyObj)
		if err != nil {
			return err
		}
		t.Method = "PUT"
		t.URL = fmt.Sprintf("%s/kvs/%s", g.BaseURL, key)
		t.Body = b
		if t.Header == nil {
			t.Header = make(map[string][]string, 1)
		}
		t.Header["Content-Type"] = []string{"application/json"}
		return nil
	}
}

func fillRandomLetters(r *rand.Rand, buf []byte) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := range buf {
		buf[i] = letters[r.Intn(len(letters))]
	}
}
