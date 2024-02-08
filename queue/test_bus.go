package queue

import (
	"context"
	"time"
)

type TestWriteQueue struct {
	Error error
}

func (t *TestWriteQueue) SendOne(_ context.Context, _ string, _ string, _ string, _ []byte) error {
	return t.Error
}

func (t *TestWriteQueue) ScheduleOne(_ context.Context, _ string, _ string, _ string, _ []byte, _ time.Time) error {
	return t.Error
}
