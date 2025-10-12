# Kavos

シンプル & 高速なインメモリ KVS (Go)
- ジェネリク・シャーディング (2^n shards)
- TTL (遅延削除 + 周期クリーンアップ)
- LRU Eviction (容量制限)
- 構造化ログ (slog)
- HTTP API (chi)
- 拡張容易な Evictor / Logger / Metrics (予定)

## 特徴 (Features)
- Set/Get/Delete
- TTL: ?ttl=秒 で指定 (0 / 無指定で無期限)
- バックグラウンド TTL クリーン (interval 可変)
- LRU 方式 Eviction (容量超過時に古いキーを追い出し)
- シャーディングによるロック競合削減
- Lazy expiration + Periodic cleanup のハイブリッド
- 構造化アクセスログ / Store 内部イベントログ

## クイックスタート
```bash
go run ./cmd/server
# TTL 付きセット (5秒)
curl -X PUT 'http://localhost:8080/kvs/hello?ttl=5' \
  -H 'Content-Type: application/json' -d '{"value":"world"}'
curl 'http://localhost:8080/kvs/hello'
sleep 6
curl -i 'http://localhost:8080/kvs/hello'  # 404
```

## HTTP API
| Method | Path            | 説明                        | 備考 |
|--------|-----------------|-----------------------------|------|
| PUT    | /kvs/{key}      | 値を設定 (JSON: {"value"})  | ?ttl=秒 |
| GET    | /kvs/{key}      | 値を取得                    | 404=未存在/期限切れ |
| DELETE | /kvs/{key}      | 削除                        |      |
| GET    | /healthz (任意) | 健康チェック (追加予定)     |      |

Request (PUT):
```json
{"value":"foo"}
```
Response (成功例):
```json
{"key":"hello","value":"world"}
```
エラーは:
```json
{"error":"message"}
```

## コード使用例
```go
logger := log.New()
st := store.New[string,string](
  store.WithShards(32),
  store.WithCleanupInterval(2*time.Second),
  store.WithLogger(logger),
).WithEvictor(store.NewLRUEvictor[string,string](10000))

h := http.NewRouter(st, logger)
```

## オプション
- WithShards(n) : シャード数 (2 の冪に繰上)
- WithCleanupInterval(d) : TTL クリーン周期間隔 (0=無効)
- WithLogger(l) : 構造化ログ出力
- WithEvictor(ev) : Eviction ポリシー (例: LRU)

## LRU Eviction
```go
st.WithEvictor(store.NewLRUEvictor[string,string](capacity))
```
capacity 超過で最も古い (低頻度) キーを削除。

## TTL
- PUT /kvs/key?ttl=5 で 5 秒後に期限
- アクセス時に期限切れなら遅延削除
- cleanup interval により非アクセスでも削除

## ログ
環境変数:
```
LOG_LEVEL=debug
```
出力例:
```
http.access method=GET path=/kvs/hello status=200 duration_ms=1 bytes=34
store.set key=hello ttl=5s
store.ttl.cleanup shard=3 removed=12
store.evict count=1 victims=[oldkey]
```

## 開発
```bash
go test -race ./...
go test -bench=. ./internal/store
```

## 今後のロードマップ
- Prometheus メトリクス (ヒット率 / エビクション数 / TTL 削除件数)
- Request ID / TraceID ミドルウェア
- Config ファイル / Flags
- 他 Eviction: LFU, TinyLFU
- Snapshots / Persistence (オプション)

