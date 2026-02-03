package tick

import (
	"context"
	"sync"
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

	// mu protects running state.
	mu sync.Mutex

	// wg waits for the tick loop to finish.
	wg sync.WaitGroup

	// onTick is called after each tick completes (for external hooks).
	onTick func(tickNumber int64)
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

	// Advance game time
	t.gameTime.Advance()

	// Record metrics
	duration := time.Since(startTime)
	t.metrics.Record(duration)

	// Track if we're falling behind
	if duration > TickDuration {
		t.metrics.TicksBehind++
	}

	// Call tick callback if set
	if t.onTick != nil {
		t.onTick(t.gameTime.TotalTicks)
	}
}

// Stop gracefully stops the tick loop.
func (t *Ticker) Stop() {
	t.cancel()
	t.wg.Wait()
}

// IsRunning returns whether the ticker is currently running.
func (t *Ticker) IsRunning() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running
}

// Scheduler returns the tick scheduler.
func (t *Ticker) Scheduler() *Scheduler {
	return t.scheduler
}

// GameTime returns the current game time.
func (t *Ticker) GameTime() *GameTime {
	return t.gameTime
}

// Metrics returns the tick metrics.
func (t *Ticker) Metrics() *TickMetrics {
	return t.metrics
}

// SetOnTick sets a callback function to be called after each tick.
func (t *Ticker) SetOnTick(callback func(tickNumber int64)) {
	t.onTick = callback
}

// CurrentTick returns the current tick number.
func (t *Ticker) CurrentTick() int64 {
	return t.gameTime.TotalTicks
}

// TPS returns the current ticks per second.
func (t *Ticker) TPS() float64 {
	return t.metrics.TPS()
}
