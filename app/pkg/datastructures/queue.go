package datastructures

import "fmt"

type Equaler[T any] interface {
	Equals(T) bool
}

type Queue[T Equaler[T]] interface {
	Pop() (T, error)
	Push(elem T)
	Remove(elem T) bool
	Len() int
	Clean()
}

type fifoQueue[T Equaler[T]] struct {
	values []T
}

func NewFifoQueue[T Equaler[T]]() *fifoQueue[T] {
	return &fifoQueue[T]{}
}

func (q *fifoQueue[T]) Pop() (T, error) {
	var val T
	if len(q.values) == 0 {
		return val, fmt.Errorf("no elements in queue")
	}

	val = q.values[len(q.values)-1]
	q.values = q.values[0 : len(q.values)-1]
	return val, nil
}

func (q *fifoQueue[T]) Push(elem T) {
	if len(q.values) == 0 {
		q.values = append(q.values, elem)
		return
	}

	var prev T
	for i, element := range q.values {
		if i == 0 {
			q.values[0] = elem
		} else {
			q.values[i] = prev
		}

		prev = element
	}

	q.values = append(q.values, prev)
}

func (q *fifoQueue[T]) Len() int {
	return len(q.values)
}

func (q *fifoQueue[T]) Remove(elem T) bool {
	idx := q.search(elem)
	if idx < 0 {
		return false
	}

	q.values = append(q.values[0:idx], q.values[idx+1:len(q.values)]...)
	return true
}

func (q *fifoQueue[T]) Clean() {
	q.values = q.values[:0]
}

func (q *fifoQueue[T]) search(elem T) int {
	for i, v := range q.values {
		if elem.Equals(v) {
			return i
		}
	}
	return -1
}
