package store

import (
	"fmt"
	"hash/fnv"
)

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
