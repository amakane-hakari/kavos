package store

import (
	"container/list"
	"sync"
)

// LRUEvictor は Store 全体で LRU を 1つ保持（シャード跨ぎ）
type LRUEvictor[K comparable, V any] struct {
	cap int
	mu  sync.Mutex
	ll  *list.List          // Front = 最も古い（victim）, Back = 最近使用
	idx map[K]*list.Element // key -> *Element
}

type lruItem[K comparable] struct {
	key K
}

// NewLRUEvictor は新しい LRUEvictor を作成します。
func NewLRUEvictor[K comparable, V any](capacity int) *LRUEvictor[K, V] {
	if capacity <= 0 {
		capacity = 1 // 最低でも1つは保持
	}
	return &LRUEvictor[K, V]{
		cap: capacity,
		ll:  list.New(),
		idx: make(map[K]*list.Element),
	}
}

// Size は現在のサイズを返します。
func (l *LRUEvictor[K, V]) Size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ll.Len()
}

// OnSet はアイテムがセットされたときに呼び出されます。
func (l *LRUEvictor[K, V]) OnSet(key K, _ V, existed bool) (victims []K) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if existed {
		// 既存のアイテムを更新
		if el, ok := l.idx[key]; ok {
			l.ll.MoveToBack(el)
			return nil
		}
	}

	// 新しいアイテムを追加
	el := l.ll.PushBack(&lruItem[K]{key: key})
	l.idx[key] = el

	// キャパシティを超えた場合は最も古いアイテムを削除
	for l.ll.Len() > l.cap {
		front := l.ll.Front()
		it := front.Value.(*lruItem[K])
		victims = append(victims, it.key)
		delete(l.idx, it.key)
		l.ll.Remove(front)
	}

	return victims
}

// OnGet はアイテムが取得されたときに呼び出されます。
func (l *LRUEvictor[K, V]) OnGet(key K, hit bool) {
	if !hit {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.idx[key]; ok {
		l.ll.MoveToBack(el)
	}
}

// OnDelete はアイテムが削除されたときに呼び出されます。
func (l *LRUEvictor[K, V]) OnDelete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.idx[key]; ok {
		delete(l.idx, key)
		l.ll.Remove(el)
	}
}
