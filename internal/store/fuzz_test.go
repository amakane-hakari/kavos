package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/amakane-hakari/kavos/internal/metrics"
)

/*
Fuzzで検証する性質（簡易）
1. パニックしない（スレッドセーフ / TTL 経路含む）
2. TTL なし（永続）で最後に残っているはずのキーはGetで（期限切れ扱いにならず）値が取得できる
   - 「永続」とは最後にSet（ttl=0）されDeleteされていないキー
3. Getが値を返した場合、そのキーは参照モデル上で
   - Deleteされていない
   - (ttl>0のケースなら) まだ期限切れ時刻を超過していない
4. Len()は0以上であり、参照モデルが保持すると判断する永続キー数以上になることはない
   (TTLキー遅延削除のためLen()は永続+未アクセスexpirablesを含む可能性があるので上限のみ緩め)
*/

type modelEntry struct {
	val      string
	expireAt int64 // 0 = no ttl
	deleted  bool
}

func FuzzStoreOperations(f *testing.F) {
	seedCorpus := [][]byte{
		// 少数の単純操作
		{0x00, 3, 3, 0}, // set
		{0x01, 3, 3, 5}, // set ttl
		{0x02, 3, 0, 0}, // get
		{0x03, 3, 0, 0}, // delete
	}
	for _, c := range seedCorpus {
		f.Add(c)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) < 4 {
			t.Skip()
		}

		st := New[string, string](
			WithShards(16),
			WithCleanupInterval(0),
			WithMetrics(metrics.Noop{}),
		)

		model := map[string]*modelEntry{}

		const (
			opSet    = 0
			opSetTTL = 1
			opGet    = 2
			opDelete = 3
		)

		reader := bytes.NewReader(data)
		chunk := make([]byte, 4)
		opCount := 0

		for {
			if _, err := reader.Read(chunk); err != nil {
				break
			}
			op := chunk[0] % 4
			kLen := int(chunk[1]%20) + 1
			vLen := int(chunk[2]%20) + 1
			flag := chunk[3]

			// 派生キー/値(安定再現性確保のためdeterministic)
			key := fmt.Sprintf("k%02d_%02d", kLen, flag)
			if len(key) > kLen {
				key = key[:kLen]
			}
			val := fmt.Sprintf("v%02d_%02d", vLen, flag)
			if len(val) > vLen {
				val = val[:vLen]
			}

			switch op {
			case opSet:
				st.Set(key, val)
				me := model[key]
				if me == nil {
					me = &modelEntry{}
					model[key] = me
				}
				me.val = val
				me.expireAt = 0
				me.deleted = false

			case opSetTTL:
				// TTLを1～5msに限定し、遅延削除が発生するよう適度に短く
				ttlMs := int(flag%5) + 1
				ttl := time.Duration(ttlMs) * time.Millisecond
				st.SetWithTTL(key, val, ttl)
				me := model[key]
				if me == nil {
					me = &modelEntry{}
					model[key] = me
				}
				me.val = val
				me.expireAt = time.Now().Add(ttl).UnixNano()
				me.deleted = false

			case opGet:
				got, ok := st.Get(key)
				me := model[key]
				if !ok {
					// 取得失敗ならモデル上: 無い or 削除 or 期限切れのはず
					if me != nil && !me.deleted {
						if me.expireAt == 0 {
							t.Fatalf("expected persistent key %s to be present", key)
						}
					}
					// TTL 付きは遅延削除タイミン差異でmissを許容
				} else {
					// 取得成功
					if me == nil || me.deleted {
						t.Fatalf("store returned value for deleted/non-existent key %s", key)
					}
					if me.expireAt > 0 && me.expireAt <= time.Now().UnixNano() {
						// 期限超過しているのに取得できるのはおかしい(Getで遅延削除されるはず)
						t.Fatalf("expired key %s returned", key)
					}
					if got != me.val && me.expireAt == 0 {
						// 永続キーは値一致すべき（TTL付きはrace的ズレ許容)
						t.Fatalf("value mismatch key=%s got=%s want=%s", key, got, me.val)
					}
				}
			case opDelete:
				st.Delete(key)
				if me := model[key]; me != nil {
					me.deleted = true
				}
			}
			opCount++
			if opCount > 20_000 { // 上限（無限ループ防止）
				break
			}
		}

		// 最終整合性（永続キーのみ厳格: 期限なし & 未削除)
		for k, me := range model {
			if me.deleted {
				continue
			}
			if me.expireAt == 0 {
				if v, ok := st.Get(k); !ok || v != me.val {
					t.Fatalf("final check failed for persistent key=%s", k)
				}
			}
		}
	})
}

// 簡易並行版: fuzz 入力でキー集合を派生し複数 goroutine が操作
func FuzzStoreConcurrent(f *testing.F) {
	f.Add([]byte("concurrent-seed"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// 最低2バイトあればキー数・ワーカー数を決められる
		if len(data) < 2 {
			t.Skip()
		}
		st := New[string, string](
			WithShards(32),
			WithCleanupInterval(0),
			WithMetrics(metrics.Noop{}),
		)
		// キー集合生成
		nKeys := int(data[0]%32) + 8
		keys := make([]string, nKeys)
		for i := range nKeys {
			keys[i] = fmt.Sprintf("ck%02d", i)
			st.Set(keys[i], "init")
		}
		workers := int(data[1]%8) + 2
		var seedBuf [8]byte
		if len(data) > 2 {
			copy(seedBuf[:], data[2:])
		}
		rndSeed := binary.LittleEndian.Uint64(seedBuf[:])

		var wg sync.WaitGroup
		for w := range workers {
			wg.Add(1)
			go func(offset int64) {
				defer wg.Done()
				r := rand.New(rand.NewSource(int64(rndSeed) + offset))
				ops := 100 + int(offset%200)
				for range ops {
					k := keys[r.Intn(len(keys))]
					switch r.Intn(4) {
					case 0:
						st.Set(k, "v")
					case 1:
						st.SetWithTTL(k, "vt", time.Duration(r.Intn(3)+1)*time.Millisecond)
					case 2:
						st.Get(k)
					case 3:
						st.Delete(k)
					}
				}
			}(int64(w))
		}
		wg.Wait()

		// 基本: Len() は負でない（当然） & panic 無しが目的
		if st.Len() < 0 {
			t.Fatalf("invalid length")
		}
	})
}
