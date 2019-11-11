// This package provides a priority queue implementation and
// scaffold interfaces.
package pqueue

import (
	"container/heap"
	"errors"
	"sync"
)

// Only items implementing this interface can be enqueued
// on the priority queue.
type Interface interface {
	Less(other interface{}) bool
	Index() int
	UpdateIndex(i int)
}

// Queue is a threadsafe priority queue exchange. Here's
// a trivial example of usage:
//
//     q := pqueue.New(0)
//     go func() {
//         for {
//             task := q.Dequeue()
//             println(task.(*CustomTask).Name)
//         }
//     }()
//     for i := 0; i < 100; i := 1 {
//         task := CustomTask{Name: "foo", priority: rand.Intn(10)}
//         q.Enqueue(&task)
//     }
//
type Queue struct {
	Limit int
	items *sorter
	cond  *sync.Cond
}

// New creates and initializes a new priority queue, taking
// a limit as a parameter. If 0 given, then queue will be
// unlimited.
func New(max int) (q *Queue) {
	var locker sync.Mutex
	q = &Queue{Limit: max}
	q.items = new(sorter)
	q.cond = sync.NewCond(&locker)
	heap.Init(q.items)
	return
}

// Enqueue puts given item to the queue.
func (q *Queue) Enqueue(item Interface) error {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	if q.Limit > 0 && q.Len() >= q.Limit {
		return errors.New("queue limit reached")
	}
	heap.Push(q.items, item)
	q.cond.Signal()
	return nil
}

// Dequeue takes an item from the queue. If queue is empty
// then should block waiting for at least one item.
func (q *Queue) Dequeue() Interface {
	q.cond.L.Lock()
start:
	x := heap.Pop(q.items)
	if x == nil {
		q.cond.Wait()
		goto start
	}
	q.cond.L.Unlock()
	return x.(Interface)
}

func (q *Queue) Front() Interface {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.items.Front().(Interface)
}

func (q *Queue) Back() Interface {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	return q.items.Back().(Interface)
}

// Remove removes the element at index i from the heap.
// The complexity is O(log n) where n = h.Len().
func (q *Queue) Remove(item Interface) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.items.Remove(item.Index())
}

// Safely changes enqueued items limit. When limit is set
// to 0, then queue is unlimited.
func (q *Queue) ChangeLimit(newLimit int) {
	q.cond.L.Lock()
	defer q.cond.L.Unlock()
	q.Limit = newLimit
}

// Len returns number of enqueued elemnents.
func (q *Queue) Len() int {
	return q.items.Len()
}

type sorter []Interface

func (s *sorter) Push(i interface{}) {
	n := len(*s)
	item, ok := i.(Interface)
	if !ok {
		return
	}
	item.UpdateIndex(n)
	*s = append(*s, item)
}

func (s *sorter) Pop() interface{} {
	old := *s
	n := len(old)
	if n > 0 {
		item := old[n-1]
		old[n-1] = nil       // avoid memory leak
		item.UpdateIndex(-1) // for safety
		*s = old[0 : n-1]
		return item
	}
	return nil
}

func (s *sorter) Remove(i int) {
	heap.Remove(s, i)
}

func (s *sorter) Front() interface{} {
	if s.Len() > 0 {
		return (*s)[0]
	}
	return nil
}

func (s *sorter) Back() interface{} {
	n := s.Len()
	if n > 0 {
		return (*s)[n-1]
	}
	return nil
}

func (s sorter) Len() int { return len(s) }

func (s sorter) Less(i, j int) bool { return s[i].Less(s[j]) }

func (s sorter) Swap(i, j int) {
	if s.Len() > 0 {
		s[i], s[j] = s[j], s[i]
		s[i].UpdateIndex(i)
		s[j].UpdateIndex(j)
	}
}
