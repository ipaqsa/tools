package queue

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gammazero/deque"

	queuetask "tools/queue/task"
)

type Queue struct {
	wg   *sync.WaitGroup
	name string

	ctx    context.Context
	cancel context.CancelFunc

	completed []string

	once sync.Once

	mu    sync.Mutex
	deque deque.Deque[*queuetask.Task]

	handler Handler
}

type Handler func(context.Context, *queuetask.Task) error

func newQueue(name string, handler Handler) *Queue {
	return &Queue{
		wg:        new(sync.WaitGroup),
		name:      name,
		handler:   handler,
		completed: []string{},
		deque:     deque.Deque[*queuetask.Task]{},
	}
}

func (q *Queue) Enqueue(task *queuetask.Task) string {
	if q == nil || task == nil {
		return ""
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	task.Set("queue", q.name)
	q.deque.PushBack(task)

	return task.ID()
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

	q.mu.Lock()
	q.completed = append(q.completed, t.ID())
	if len(q.completed) > 100 {
		q.completed = q.completed[80:]
	}
	q.mu.Unlock()
}

func (q *Queue) Wait(wg *sync.WaitGroup, task *queuetask.Task) {
	q.Enqueue(task)

	go func() {
		wg.Add(1)
		defer wg.Done()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-q.ctx.Done():
				return
			case <-ticker.C:
				q.mu.Lock()
				if slices.Contains(q.completed, task.ID()) {
					q.mu.Unlock()
					return
				}
				q.mu.Unlock()
			}
		}
	}()
}

func (q *Queue) Stop() {
	q.cancel()
	q.wg.Wait()
}

func (q *Queue) Snapshots() []*queuetask.Task {
	q.mu.Lock()
	defer q.mu.Unlock()

	return slices.Collect(q.deque.Iter())
}
