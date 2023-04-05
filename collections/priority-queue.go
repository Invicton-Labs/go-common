package collections

import (
	"container/heap"

	"github.com/Invicton-Labs/go-common/constraints"
)

// Source: example from pkg.go.dev/container/heap

// A PriorityQueue is a data structure that pops elements in order of
// descending priority. It's closer to a self-sorting Stack.
type PriorityQueue[T any, P constraints.Ordered] interface {
	// Push adds a new item to the PriorityQueue, automatically inserting it
	// at the right position based on its priority value.
	Push(T, P)
	// Pop retrieves the item with the highest priority value. If the queue is
	// empty, it will panic (always check with Empty() first).
	Pop() (T, P)
	// Len will return the number of items in the queue.
	Len() int
	// Empty will return true if there are no items remaining in the queue.
	Empty() bool
}

// An Item is something we manage in a priority queue.
type priorityQueueItem[T any, P constraints.Ordered] struct {
	value    T // The value of the item; arbitrary.
	priority P // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

type priorityQueueInternal[T any, P constraints.Ordered] []*priorityQueueItem[T, P]

func (pq priorityQueueInternal[T, P]) Len() int {
	return len(pq)
}

func (pq priorityQueueInternal[T, P]) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].priority > pq[j].priority
}

// Swap swaps two elements in the queue
func (pq priorityQueueInternal[T, P]) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueueInternal[T, P]) Push(x any) {
	n := len(*pq)
	item := x.(*priorityQueueItem[T, P])
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueueInternal[T, P]) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

type priorityQueue[T any, P constraints.Ordered] struct {
	pq priorityQueueInternal[T, P]
}

func NewPriorityQueue[T any, P constraints.Ordered]() PriorityQueue[T, P] {
	pq := make(priorityQueueInternal[T, P], 0)
	heap.Init(&pq)
	return &priorityQueue[T, P]{
		pq: pq,
	}
}

func (pq *priorityQueue[T, P]) Push(value T, priority P) {
	item := &priorityQueueItem[T, P]{
		value:    value,
		priority: priority,
	}
	heap.Push(&pq.pq, item)
}

func (pq *priorityQueue[T, P]) Pop() (value T, priority P) {
	item := heap.Pop(&pq.pq).(*priorityQueueItem[T, P])
	return item.value, item.priority
}

func (pq priorityQueue[T, P]) Len() int {
	return pq.pq.Len()
}

func (pq priorityQueue[T, P]) Empty() bool {
	return pq.pq.Len() != 0
}
