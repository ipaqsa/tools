package task

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

type Task struct {
	kind    string
	subject string
	meta    metadata

	backoff   backoff.BackOff
	nextRetry time.Time
}

type metadata map[string]string

func New(kind, subject string) *Task {
	return &Task{
		kind:      kind,
		subject:   subject,
		meta:      make(map[string]string),
		backoff:   backoff.NewExponentialBackOff(),
		nextRetry: time.Now(),
	}
}

func (t *Task) Get(key string) string {
	if t.meta == nil {
		return ""
	}

	return t.meta[key]
}

func (t *Task) Set(key, value string) {
	if t.meta == nil {
		t.meta = make(map[string]string)
	}

	t.meta[key] = value
}

func (t *Task) Delete(key string) {
	if t.meta == nil {
		return
	}

	delete(t.meta, key)
}

func (t *Task) Kind() string {
	return t.kind
}

func (t *Task) Subject() string {
	return t.subject
}

func (t *Task) Backoff() backoff.BackOff {
	return t.backoff
}

func (t *Task) NextRetry() time.Time {
	return t.nextRetry
}

func (t *Task) SetNextRetry(nextRetry time.Time) {
	t.nextRetry = nextRetry
}
