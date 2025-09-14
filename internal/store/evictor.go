package store

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
