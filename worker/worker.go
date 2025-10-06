package worker

import (
	"context"
)

type Worker struct {
	queues map[string]*Queue
}

func New() *Worker {
	return &Worker{
		queues: make(map[string]*Queue),
	}
}

func (w *Worker) Get(name string) *Queue {
	return w.queues[name]
}

func (w *Worker) Spawn(ctx context.Context, name string, handler Handler) {
	if _, ok := w.queues[name]; !ok {
		w.queues[name] = newQueue(name, handler).Start(ctx)
	}
}

func (w *Worker) Stop(name string) {
	if q := w.queues[name]; q != nil {
		q.Stop()
		delete(w.queues, name)
	}
}

func (w *Worker) StopAll() {
	for _, q := range w.queues {
		q.Stop()
		delete(w.queues, q.name)
	}
}
