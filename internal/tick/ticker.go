package tick

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Ticker manages the game tick loop.
// It ensures ticks run at a consistent rate (20 TPS) and provides
// mechanisms for starting, stopping, and monitoring the tick loop.
type Ticker struct {
	// scheduler handles phase execution order.
	scheduler *Scheduler

	// gameTime tracks the current game time.
	gameTime *GameTime

	// metrics tracks tick performance.
	metrics *TickMetrics

	// running indicates if the ticker is currently running.
	running bool

	// ctx is the context for cancellation.
	ctx context.Context

	// cancel cancels the context.
	cancel context.CancelFunc

	// mu protects running state and gameTime/metrics access.
	mu sync.RWMutex

	// wg waits for the tick loop to finish.
	wg sync.WaitGroup

	// onTick is called after each tick completes (for external hooks).
	onTick atomic.Pointer[func(tickNumber int64)]
}

// NewTicker creates a new Ticker with default settings.
func NewTicker() *Ticker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Ticker{
		scheduler: NewScheduler(),
		gameTime:  NewGameTime(),
		metrics:   NewTickMetrics(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins the tick loop.
// This method blocks until Stop is called or the context is cancelled.
// For non-blocking operation, call this in a goroutine.
func (t *Ticker) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running = true
	t.wg.Add(1)
	t.mu.Unlock()

	defer t.wg.Done()

	ticker := time.NewTicker(TickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-t.ctx.Done():
			t.mu.Lock()
			t.running = false
			t.mu.Unlock()
			return

		case tickTime := <-ticker.C:
			t.executeTick(tickTime)
		}
	}
}

// executeTick performs a single game tick.
func (t *Ticker) executeTick(tickTime time.Time) {
	startTime := time.Now()

	// Execute all phases in order
	t.scheduler.ExecuteAllPhases()

	// Advance game time and record metrics (protected by mutex)
	t.mu.Lock()
	t.gameTime.Advance()
	duration := time.Since(startTime)
	t.metrics.Record(duration)
	if duration > TickDuration {
		t.metrics.TicksBehind++
	}
	totalTicks := t.gameTime.TotalTicks
	t.mu.Unlock()

	// Call tick callback if set
	if callback := t.onTick.Load(); callback != nil {
		(*callback)(totalTicks)
	}
}

// Stop gracefully stops the tick loop.
func (t *Ticker) Stop() {
	t.cancel()
	t.wg.Wait()
}

// IsRunning returns whether the ticker is currently running.
func (t *Ticker) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// Scheduler returns the tick scheduler.
func (t *Ticker) Scheduler() *Scheduler {
	return t.scheduler
}

// GameTime returns a copy of the current game time.
// Note: Returns a snapshot; the actual time continues advancing.
func (t *Ticker) GameTime() GameTime {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return *t.gameTime
}

// Metrics returns a copy of the current tick metrics.
// Note: Returns a snapshot; metrics continue updating.
func (t *Ticker) Metrics() TickMetrics {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return *t.metrics
}

// SetOnTick sets a callback function to be called after each tick.
func (t *Ticker) SetOnTick(callback func(tickNumber int64)) {
	t.onTick.Store(&callback)
}

// CurrentTick returns the current tick number.
func (t *Ticker) CurrentTick() int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.gameTime.TotalTicks
}

// TPS returns the current ticks per second.
func (t *Ticker) TPS() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.metrics.TPS()
}
