package queue

import (
	"context"
)

type Manager struct {
	queues map[string]*Queue
}

func New() *Manager {
	return &Manager{
		queues: make(map[string]*Queue),
	}
}

func (m *Manager) Get(name string) *Queue {
	return m.queues[name]
}

func (m *Manager) Spawn(ctx context.Context, name string, handler Handler) {
	if _, ok := m.queues[name]; !ok {
		m.queues[name] = newQueue(name, handler).Start(ctx)
	}
}

func (m *Manager) Stop(name string) {
	if q := m.queues[name]; q != nil {
		q.Stop()
		delete(m.queues, name)
	}
}

func (m *Manager) StopAll() {
	for _, q := range m.queues {
		q.Stop()
		delete(m.queues, q.name)
	}
}
