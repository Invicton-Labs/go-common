// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package list implements a doubly linked list.
//
// To iterate over a list (where l is a *List):
//
//	for e := l.Front(); e != nil; e = e.Next() {
//		// do something with e.Value
//	}
package collections

type LinkedListElement[T any] interface {
	List() LinkedList[T]
	Prev() LinkedListElement[T]
	Next() LinkedListElement[T]
	Value() T
	setNext(element LinkedListElement[T])
	setPrev(element LinkedListElement[T])
	setList(list *linkedList[T])
}

// linkedListElement[T] is an element of a linked list.
type linkedListElement[T any] struct {
	// Next and previous pointers in the doubly-linked list of elements.
	// To simplify the implementation, internally a list l is implemented
	// as a ring, such that &l.root is both the next element of the last
	// list element (l.Back()) and the previous element of the first list
	// element (l.Front()).
	nextElement, prevElement LinkedListElement[T]

	// The list to which this element belongs.
	list *linkedList[T]

	// The value stored with this element.
	value T
}

// list returns the list that the element belongs to

func (e *linkedListElement[T]) List() LinkedList[T] {
	return e.list
}

func (e *linkedListElement[T]) setNext(element LinkedListElement[T]) {
	e.nextElement = element
}
func (e *linkedListElement[T]) setPrev(element LinkedListElement[T]) {
	e.prevElement = element
}
func (e *linkedListElement[T]) setList(list *linkedList[T]) {
	e.list = list
}

// Next returns the next list element or nil.
func (e *linkedListElement[T]) Next() LinkedListElement[T] {
	if p := e.nextElement; e.list != nil && p != e.list.Front().Prev() {
		return p
	}
	return nil
}

// Prev returns the previous list element or nil.
func (e *linkedListElement[T]) Prev() LinkedListElement[T] {
	if p := e.prevElement; e.list != nil && p != e.list.root {
		return p
	}
	return nil
}

// Prev returns the previous list element or nil.
func (e *linkedListElement[T]) Value() T {
	return e.value
}

type LinkedList[T any] interface {
	// Init initializes or clears list l.
	Init() LinkedList[T]
	// Len returns the number of elements of list l.
	// The complexity is O(1).
	Len() int
	// Front returns the first element of list l or nil if the list is empty.
	Front() LinkedListElement[T]
	// Back returns the last element of list l or nil if the list is empty.
	Back() LinkedListElement[T]
	// Remove removes e from l if e is an element of list l.
	// It returns the element value e.Value.
	// The element must not be nil.
	Remove(e LinkedListElement[T]) T
	// PushFront inserts a new element e with value v at the front of list l and returns e.
	PushFront(v T) LinkedListElement[T]
	// PushBack inserts a new element e with value v at the back of list l and returns e.
	PushBack(v T) LinkedListElement[T]
	// InsertBefore inserts a new element e with value v immediately before mark and returns e.
	// If mark is not an element of l, the list is not modified.
	// The mark must not be nil.
	InsertBefore(v T, mark LinkedListElement[T]) LinkedListElement[T]
	// InsertAfter inserts a new element e with value v immediately after mark and returns e.
	// If mark is not an element of l, the list is not modified.
	// The mark must not be nil.
	InsertAfter(v T, mark LinkedListElement[T]) LinkedListElement[T]
	// MoveToFront moves element e to the front of list l.
	// If e is not an element of l, the list is not modified.
	// The element must not be nil.
	MoveToFront(e LinkedListElement[T])
	// MoveToBack moves element e to the back of list l.
	// If e is not an element of l, the list is not modified.
	// The element must not be nil.
	MoveToBack(e LinkedListElement[T])
	// MoveBefore moves element e to its new position before mark.
	// If e or mark is not an element of l, or e == mark, the list is not modified.
	// The element and mark must not be nil.
	MoveBefore(e, mark LinkedListElement[T])
	// MoveAfter moves element e to its new position after mark.
	// If e or mark is not an element of l, or e == mark, the list is not modified.
	// The element and mark must not be nil.
	MoveAfter(e, mark LinkedListElement[T])
	// PushBackList inserts a copy of another list at the back of list l.
	// The lists l and other may be the same. They must not be nil.
	PushBackList(other LinkedList[T])
	// PushFrontList inserts a copy of another list at the front of list l.
	// The lists l and other may be the same. They must not be nil.
	PushFrontList(other LinkedList[T])
}

// linkedList[T] represents a doubly linked list.
// The zero value for linkedList[T] is an empty list ready to use.
type linkedList[T any] struct {
	root *linkedListElement[T] // sentinel list element, only &root, root.prev, and root.next are used
	len  int                   // current list length excluding (this) sentinel element
}

func (l *linkedList[T]) Init() LinkedList[T] {
	l.root.setNext(l.root)
	l.root.setPrev(l.root)
	l.len = 0
	return l
}

// New returns an initialized list.
func NewLinkedList[T any]() LinkedList[T] {
	return new(linkedList[T]).Init()
}

// Len returns the number of elements of list l.
// The complexity is O(1).
func (l *linkedList[T]) Len() int { return l.len }

// Front returns the first element of list l or nil if the list is empty.
func (l *linkedList[T]) Front() LinkedListElement[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.Next()
}

// Back returns the last element of list l or nil if the list is empty.
func (l *linkedList[T]) Back() LinkedListElement[T] {
	if l.len == 0 {
		return nil
	}
	return l.root.Prev()
}

// lazyInit lazily initializes a zero List[T] value.
func (l *linkedList[T]) lazyInit() {
	if l.root.Next() == nil {
		l.Init()
	}
}

// insert inserts e after at, increments l.len, and returns e.
func (l *linkedList[T]) insert(e, at LinkedListElement[T]) LinkedListElement[T] {
	e.setPrev(at)
	e.setNext(at.Next())
	e.Prev().setNext(e)
	e.Next().setPrev(e)
	e.setList(l)
	l.len++
	return e
}

// insertValue is a convenience wrapper for insert(&Element{Value: v}, at).
func (l *linkedList[T]) insertValue(v T, at LinkedListElement[T]) LinkedListElement[T] {
	return l.insert(&linkedListElement[T]{value: v}, at)
}

// remove removes e from its list, decrements l.len
func (l *linkedList[T]) remove(e LinkedListElement[T]) {
	e.Prev().setNext(e.Next())
	e.Next().setPrev(e.Prev())
	e.setNext(nil) // avoid memory leaks
	e.setPrev(nil) // avoid memory leaks
	e.setList(nil)
	l.len--
}

// move moves e to next to at.
func (l *linkedList[T]) move(e, at LinkedListElement[T]) {
	if e == at {
		return
	}
	e.Prev().setNext(e.Next())
	e.Next().setPrev(e.Prev())
	e.setPrev(at)
	e.setNext(at.Next())
	e.Prev().setNext(e)
	e.Next().setPrev(e)
}

// Remove removes e from l if e is an element of list l.
// It returns the element value e.Value.
// The element must not be nil.
func (l *linkedList[T]) Remove(e LinkedListElement[T]) T {
	if e.List() == l {
		// if e.list == l, l must have been initialized when e was inserted
		// in l or l == nil (e is a zero Element[T]) and l.remove will crash
		l.remove(e)
	}
	return e.Value()
}

// PushFront inserts a new element e with value v at the front of list l and returns e.
func (l *linkedList[T]) PushFront(v T) LinkedListElement[T] {
	l.lazyInit()
	return l.insertValue(v, l.root)
}

// PushBack inserts a new element e with value v at the back of list l and returns e.
func (l *linkedList[T]) PushBack(v T) LinkedListElement[T] {
	l.lazyInit()
	return l.insertValue(v, l.root.Prev())
}

// InsertBefore inserts a new element e with value v immediately before mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *linkedList[T]) InsertBefore(v T, mark LinkedListElement[T]) LinkedListElement[T] {
	if mark.List() != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark.Prev())
}

// InsertAfter inserts a new element e with value v immediately after mark and returns e.
// If mark is not an element of l, the list is not modified.
// The mark must not be nil.
func (l *linkedList[T]) InsertAfter(v T, mark LinkedListElement[T]) LinkedListElement[T] {
	if mark.List() != l {
		return nil
	}
	// see comment in List.Remove about initialization of l
	return l.insertValue(v, mark)
}

// MoveToFront moves element e to the front of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *linkedList[T]) MoveToFront(e LinkedListElement[T]) {
	if e.List() != l || l.root.Next() == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, l.root)
}

// MoveToBack moves element e to the back of list l.
// If e is not an element of l, the list is not modified.
// The element must not be nil.
func (l *linkedList[T]) MoveToBack(e LinkedListElement[T]) {
	if e.List() != l || l.root.Prev() == e {
		return
	}
	// see comment in List.Remove about initialization of l
	l.move(e, l.root.Prev())
}

// MoveBefore moves element e to its new position before mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
func (l *linkedList[T]) MoveBefore(e, mark LinkedListElement[T]) {
	if e.List() != l || e == mark || mark.List() != l {
		return
	}
	l.move(e, mark.Prev())
}

// MoveAfter moves element e to its new position after mark.
// If e or mark is not an element of l, or e == mark, the list is not modified.
// The element and mark must not be nil.
func (l *linkedList[T]) MoveAfter(e, mark LinkedListElement[T]) {
	if e.List() != l || e == mark || mark.List() != l {
		return
	}
	l.move(e, mark)
}

// PushBackList inserts a copy of another list at the back of list l.
// The lists l and other may be the same. They must not be nil.
func (l *linkedList[T]) PushBackList(other LinkedList[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Front(); i > 0; i, e = i-1, e.Next() {
		l.insertValue(e.Value(), l.root.Prev())
	}
}

// PushFrontList inserts a copy of another list at the front of list l.
// The lists l and other may be the same. They must not be nil.
func (l *linkedList[T]) PushFrontList(other LinkedList[T]) {
	l.lazyInit()
	for i, e := other.Len(), other.Back(); i > 0; i, e = i-1, e.Prev() {
		l.insertValue(e.Value(), l.root)
	}
}
