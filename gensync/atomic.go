package gensync

import (
	"sync"

	"github.com/Invicton-Labs/go-common/constraints"
)

type Atomic[T any] struct {
	l sync.Mutex
	v T
}

func NewAtomic[T any](val T) Atomic[T] {
	return Atomic[T]{
		v: val,
	}
}

func (a *Atomic[T]) Load() T {
	a.l.Lock()
	defer a.l.Unlock()
	return a.v
}

func (a *Atomic[T]) Store(val T) {
	a.l.Lock()
	defer a.l.Unlock()
	a.v = val
}

func (a *Atomic[T]) StoreIf(val T, condition func(old T, new T) bool) (stored bool) {
	a.l.Lock()
	defer a.l.Unlock()
	if condition(a.v, val) {
		a.v = val
		return true
	}
	return false
}

type AtomicComparable[T comparable] Atomic[T]

func NewAtomicComparable[T comparable](val T) AtomicComparable[T] {
	return AtomicComparable[T]{
		v: val,
	}
}

func (a *AtomicComparable[T]) CompareAndSwap(old T, new T) (swapped bool) {
	a.l.Lock()
	defer a.l.Unlock()
	if a.v == old {
		a.v = new
		return true
	}
	return false
}

type AtomicNumeric[T constraints.Numeric] AtomicComparable[T]

func NewAtomicNumeric[T constraints.Numeric](val T) AtomicNumeric[T] {
	return AtomicNumeric[T]{
		v: val,
	}
}

func (a *AtomicNumeric[T]) Add(delta T) (new T) {
	a.l.Lock()
	defer a.l.Unlock()
	a.v += delta
	return a.v
}

func (a *AtomicNumeric[T]) Subtract(delta T) (new T) {
	a.l.Lock()
	defer a.l.Unlock()
	a.v = a.v - delta
	return a.v
}
