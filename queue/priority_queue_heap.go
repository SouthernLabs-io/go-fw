package queue

// _Heap implements heap.Interface and holds Elements.
type _Heap[T any] []*Element[T]

func (h _Heap[T]) Len() int {
	return len(h)
}

func (h _Heap[T]) Less(i, j int) bool {
	eq := h[i].priority == h[j].priority
	if !eq {
		return h[i].priority < h[j].priority
	}
	return h[i].seq < h[j].seq
}

func (h _Heap[T]) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *_Heap[T]) Push(itemAny any) {
	item := itemAny.(*Element[T])
	item.index = len(*h)
	item.heap = h
	*h = append(*h, item)
}

func (h *_Heap[T]) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]

	// Remove heap reference from item
	x.index = -1
	x.heap = nil

	return x
}

func (h _Heap[T]) Peek() *Element[T] {
	return h[0]
}
