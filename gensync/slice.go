package gensync

import (
	"sync"

	"github.com/Invicton-Labs/go-common/collections"
)

type Slice[V any] interface {
	// Load returns a COPY of the slice.
	Load() (slice []V)

	// SubSlice returns a COPY of the subslice in the form s[start:end].
	SubSlice(start int, end int) (subslice []V)

	// StoreIndex will store a value in the slice at the given index.
	StoreIndex(index int, value V)

	// LoadIndex will load the value in the slice at the given index.
	LoadIndex(index int) (value V)

	// Concat will concatenate c to the end of the slice.
	Concat(c []V)

	// Append will append the value a to the end of the slice.
	Append(a V)

	// Length will get the number of elements in the slice
	Length() int
}

type slice[T any] struct {
	s []T
	l sync.Mutex
}

func NewSlice[T any](initial []T) Slice[T] {
	if initial == nil {
		initial = []T{}
	}
	return &slice[T]{
		s: initial,
	}
}

func (s *slice[V]) Load() []V {
	s.l.Lock()
	defer s.l.Unlock()
	return collections.CopySlice(s.s)
}

func (s *slice[V]) SubSlice(start int, end int) []V {
	s.l.Lock()
	defer s.l.Unlock()
	return collections.CopySlice(s.s[start:end])
}

func (s *slice[V]) StoreIndex(index int, value V) {
	s.l.Lock()
	defer s.l.Unlock()
	s.s[index] = value
}

func (s *slice[V]) LoadIndex(index int) V {
	s.l.Lock()
	defer s.l.Unlock()
	return s.s[index]
}

func (s *slice[V]) Concat(c []V) {
	s.l.Lock()
	defer s.l.Unlock()
	s.s = append(s.s, c...)
}

func (s *slice[V]) Append(a V) {
	s.l.Lock()
	defer s.l.Unlock()
	s.s = append(s.s, a)
}

func (s *slice[V]) Length() int {
	s.l.Lock()
	defer s.l.Unlock()
	return len(s.s)
}
