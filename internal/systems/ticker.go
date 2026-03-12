package systems

import (
	"context"
	"time"
)

type Ticker struct {
	ctx            context.Context
	cancel         context.CancelFunc
	handlers       []func()
	TicksPerSecond int
	tickDuration   time.Duration
}

func NewTicker(ticksPerSecond int) *Ticker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Ticker{
		ctx:            ctx,
		cancel:         cancel,
		TicksPerSecond: ticksPerSecond,
		tickDuration:   time.Second / time.Duration(ticksPerSecond),
	}
}

func (t *Ticker) Register(handler func()) {
	t.handlers = append(t.handlers, handler)
}

func (t *Ticker) executeTick() {
	for _, handler := range t.handlers {
		handler()
	}
}

func (t *Ticker) Start() {
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

			nextTickTime = nextTickTime.Add(t.tickDuration)
		}
	}
}

func (t *Ticker) Stop() {
	t.cancel()
}
