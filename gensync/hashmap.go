package gensync

import "sync"

type HashMap[T comparable] interface {
	Store(key T)
	Delete(key T)
	Has(key T) bool
	Length() int
	Keys() []T
}

type hashMap[T comparable] struct {
	m sync.Map
}

func NewHashMap[T comparable](initial []T) HashMap[T] {
	hm := &hashMap[T]{}
	for _, v := range initial {
		hm.Store(v)
	}
	return hm
}

func (lm *hashMap[T]) Store(key T) {
	lm.m.Store(key, struct{}{})
}

func (lm *hashMap[T]) Delete(key T) {
	lm.m.Delete(key)
}

func (lm *hashMap[T]) Has(key T) bool {
	_, ok := lm.m.Load(key)
	return ok
}

func (lm *hashMap[T]) Length() (length int) {
	lm.m.Range(func(key, value any) bool {
		length++
		return true
	})
	return
}

func (lm *hashMap[T]) Keys() []T {
	keys := []T{}
	lm.m.Range(func(key, value any) bool {
		keys = append(keys, key.(T))
		return true
	})
	return keys
}
