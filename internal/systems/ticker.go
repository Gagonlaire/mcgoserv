package systems

import (
	"context"
	"time"
)

const (
	TicksPerSecond = 20
	TickDuration   = time.Second / TicksPerSecond
	TicksPerDay    = 24000
)

type Handler[T any] func(ctx T)

type Ticker[T any] struct {
	handlers   []Handler[T]
	ctx        context.Context
	cancel     context.CancelFunc
	ContextVal T

	// todo: move to a handler
	TotalTicks int64
	DayTime    int64
	Day        int64
}

func NewTicker[T any](ctxVal T) *Ticker[T] {
	ctx, cancel := context.WithCancel(context.Background())
	return &Ticker[T]{
		ctx:        ctx,
		cancel:     cancel,
		ContextVal: ctxVal,
		Day:        1,
	}
}

func (t *Ticker[T]) RegisterHandler(handler Handler[T]) {
	t.handlers = append(t.handlers, handler)
}

func (t *Ticker[T]) executeTick() {
	// move this in a handler and create a world struct (for dimensions too)
	t.TotalTicks++
	t.DayTime = (t.DayTime + 1) % TicksPerDay
	if t.DayTime == 0 {
		t.Day++
	}

	for _, handler := range t.handlers {
		handler(t.ContextVal)
	}
}

func (t *Ticker[T]) Start() {
	nextTickTime := time.Now()

	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			now := time.Now()

			if now.Sub(nextTickTime) > 2*time.Second {
				nextTickTime = now
			}

			if now.Before(nextTickTime) {
				time.Sleep(nextTickTime.Sub(now))
				continue
			}

			t.executeTick()

			nextTickTime = nextTickTime.Add(TickDuration)
		}
	}
}

func (t *Ticker[T]) Stop() {
	t.cancel()
}
