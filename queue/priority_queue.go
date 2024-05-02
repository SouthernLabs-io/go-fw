package queue

import (
	"container/heap"
)

// sequence is used to keep a consistent ordering when priorities are equal.
// Overflow should not be a concern as it would take 584.9 million years to overflow running 1K tasks per second.
var sequence uint64

// Element is an element of the PriorityQueue.
type Element[T any] struct {
	// The value stored with this element.
	Value T

	// The priority of this element, it can't be changed once set.
	priority int64

	// The sequence number of this element, it is used to keep a consistent ordering when priorities are equal.
	seq uint64

	// The index of this element in the heap. It is used to remove an element in O(log(n)) instead of O(n*log(n)).
	index int

	// The heap to which this element belongs.
	heap *_Heap[T] // Pointer to the heap containing this item
}

func (i *Element[T]) Priority() int64 {
	return i.priority
}

/*
PriorityQueue this implementation is based on https://golang.org/pkg/container/heap/

High priority is closer to -infinity. Low priority is closer to +infinity.

This is not a thread-safe implementation.
*/
type PriorityQueue[T any] struct {
	h *_Heap[T]
}

func NewPriorityQueue[T any]() *PriorityQueue[T] {
	return &PriorityQueue[T]{
		h: &_Heap[T]{},
	}
}

/*
Push pushes a new element to the queue. O(log(n))

High priority is closer to -infinity. Low priority is closer to +infinity.

Two elements with the same priority are ordered by insertion order with the first element inserted being the first
returned by Pop.
*/
func (pq *PriorityQueue[T]) Push(x T, priority int64) *Element[T] {
	sequence += 1
	item := &Element[T]{Value: x, priority: priority, seq: sequence}
	heap.Push(pq.h, item)
	return item
}

// Pop removes the highest priority element from the queue and returns it. O(log(n))
func (pq *PriorityQueue[T]) Pop() *Element[T] {
	return heap.Pop(pq.h).(*Element[T])
}

// Peek returns the highest priority element without removing it. O(1)
func (pq *PriorityQueue[T]) Peek() *Element[T] {
	return pq.h.Peek()
}

/*
Remove removes an element from the queue. O(log(n))

It returns true if the element was removed, false otherwise.
*/
func (pq *PriorityQueue[T]) Remove(item *Element[T]) bool {
	if item.heap != pq.h || item.index == -1 {
		return false
	}

	heap.Remove(pq.h, item.index)
	return true
}

// Len returns the number of elements in the queue. O(1)
func (pq *PriorityQueue[T]) Len() int {
	return pq.h.Len()
}
