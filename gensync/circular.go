package gensync

import (
	"sync"
	"sync/atomic"
)

// A RingSlice is a slice that can be accessed by multiple routines
// to always return the next value in the slice, resetting at the first
// value once the last value is reached.
type RingSlice[T any] interface {
	// Next will get the next value in the slice
	Next() T
	// Set will set new values for the slice
	Set(values []T)
	// Values will get a copy of the current values
	// (not the actual internal slice, just a copy of it)
	Values() []T
}

type ringSlice[T any] struct {
	idx    atomic.Int64
	values []T
	lock   sync.RWMutex
}

func NewRingSlice[T any](values []T) RingSlice[T] {
	if len(values) == 0 {
		panic("no values provided")
	}
	idx := atomic.Int64{}
	idx.Add(-1)
	return &ringSlice[T]{
		idx:    idx,
		values: values,
	}
}

func (rs *ringSlice[T]) Next() T {
	// Only need a read lock for this operation
	rs.lock.RLock()
	defer rs.lock.RUnlock()

	// If the current index is len(values) - 1, then the
	// next value will be out of range, so reset it to -1
	rs.idx.CompareAndSwap(int64(len(rs.values))-1, -1)
	return rs.values[rs.idx.Add(1)]
}

func (rs *ringSlice[T]) Set(values []T) {
	if len(values) == 0 {
		panic("no values provided")
	}

	rs.lock.Lock()
	defer rs.lock.Unlock()

	rs.values = values
	rs.idx = atomic.Int64{}
	rs.idx.Add(-1)
}

func (rs *ringSlice[T]) Values() []T {
	dst := make([]T, len(rs.values))
	copy(dst, rs.values)
	return dst
}
