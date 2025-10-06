package queue

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gammazero/deque"

	"tools/queue/task"
)

type Queue struct {
	wg   *sync.WaitGroup
	name string

	ctx    context.Context
	cancel context.CancelFunc

	once sync.Once

	mu    sync.Mutex
	deque deque.Deque[*task.Task]

	handler Handler
}

type Handler func(context.Context, *task.Task) error

func newQueue(name string, handler Handler) *Queue {
	return &Queue{
		name:    name,
		handler: handler,
		deque:   deque.Deque[*task.Task]{},
	}
}

func (q *Queue) Enqueue(task *task.Task) {
	if q == nil || task == nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.deque.PushBack(task)
}

func (q *Queue) Start(ctx context.Context) *Queue {
	q.once.Do(func() {
		q.ctx, q.cancel = context.WithCancel(ctx)

		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					q.process(ctx)
				}
			}
		}()
	})

	return q
}

func (q *Queue) process(ctx context.Context) {
	q.mu.Lock()
	if q.deque.Len() == 0 {
		q.mu.Unlock()
		return
	}

	t := q.deque.Front()
	if t == nil {
		q.mu.Unlock()
		return
	}

	if time.Now().Before(t.NextRetry()) {
		q.mu.Unlock()
		return
	}

	q.deque.PopFront()
	q.mu.Unlock()

	if err := q.handler(ctx, t); err != nil {
		if delay := t.Backoff().NextBackOff(); delay != backoff.Stop {
			t.SetNextRetry(time.Now().Add(delay))
			q.mu.Lock()
			q.deque.PushFront(t)
			q.mu.Unlock()
		}
	}
}

func (q *Queue) Stop() {
	q.cancel()
	q.wg.Wait()
}

func (q *Queue) Wait() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.mu.Lock()
			size := q.deque.Len()
			q.mu.Unlock()
			if size == 0 {
				return
			}
		}
	}
}

func (q *Queue) Snapshots() []*task.Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	return slices.Collect(q.deque.Iter())
}
