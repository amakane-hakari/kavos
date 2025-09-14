package store

type entry[V any] struct {
	val      V
	expireAt int64 // 0 = no expiry (UnixNano)
}

const cacheLineSize = 64
