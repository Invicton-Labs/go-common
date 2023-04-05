package gensync

import (
	"sync"

	"github.com/Invicton-Labs/go-common/collections"
	"github.com/Invicton-Labs/go-common/constraints"
)

// A PriorityQueue is a data structure that pops elements in order of
// descending priority. It's closer to a self-sorting Stack. This
// version uses mutexes to ensure that it's concurrency-safe.
type PriorityQueue[T any, P constraints.Ordered] interface {
	// Push adds a new item to the PriorityQueue, automatically inserting it
	// at the right position based on its priority value.
	Push(T, P)
	// Pop retrieves the item with the highest priority value. If the queue is
	// empty, `found` will be `false`.
	Pop() (value T, priority P, found bool)
	// Len will return the number of items in the queue.
	Len() int
}

type priorityQueue[T any, P constraints.Ordered] struct {
	m  sync.Mutex
	pq collections.PriorityQueue[T, P]
}

func NewPriorityQueue[T any, P constraints.Ordered]() PriorityQueue[T, P] {
	return &priorityQueue[T, P]{
		pq: collections.NewPriorityQueue[T, P](),
	}
}

func (pq *priorityQueue[T, P]) Push(value T, priority P) {
	pq.m.Lock()
	defer pq.m.Unlock()
	pq.pq.Push(value, priority)
}

func (pq *priorityQueue[T, P]) Pop() (value T, priority P, found bool) {
	pq.m.Lock()
	defer pq.m.Unlock()
	if pq.pq.Len() == 0 {
		return value, priority, false
	}
	value, priority = pq.pq.Pop()
	return value, priority, true
}

func (pq *priorityQueue[T, P]) Len() int {
	pq.m.Lock()
	defer pq.m.Unlock()
	return pq.pq.Len()
}
