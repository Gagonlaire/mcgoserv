package tick

import (
	"sync"
	"sync/atomic"
)

// PhaseHandler is a function that handles a specific phase of the tick.
// It receives the context (typically the server) as a parameter.
type PhaseHandler[T any] func(ctx T)

// Scheduler manages the execution order of tick phases and their handlers.
type Scheduler[T any] struct {
	// handlers maps phases to their handler functions.
	handlers [PhaseCount][]PhaseHandler[T]

	// mu protects concurrent access to handlers.
	mu sync.RWMutex

	// currentPhase tracks the phase currently being executed (accessed atomically).
	currentPhase atomic.Int32

	// context is passed to all phase handlers during execution.
	context T
}

// NewScheduler creates a new tick scheduler.
func NewScheduler[T any]() *Scheduler[T] {
	s := &Scheduler[T]{
		handlers: [PhaseCount][]PhaseHandler[T]{},
	}
	s.currentPhase.Store(int32(PhaseStart))
	return s
}

// SetContext sets the context that will be passed to all phase handlers.
// This must be called before starting the ticker and should not be called
// while the ticker is running, as it may cause race conditions.
// This is typically called once during server initialization.
func (s *Scheduler[T]) SetContext(ctx T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.context = ctx
}

// Context returns the current context.
// This should only be called after SetContext has been called.
func (s *Scheduler[T]) Context() T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.context
}

// RegisterHandler registers a handler for a specific phase.
// Multiple handlers can be registered for the same phase and will be
// executed in the order they were registered.
func (s *Scheduler[T]) RegisterHandler(phase Phase, handler PhaseHandler[T]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[phase] = append(s.handlers[phase], handler)
}

// UnregisterAllHandlers removes all handlers for a specific phase.
func (s *Scheduler[T]) UnregisterAllHandlers(phase Phase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[phase] = nil
}

// ExecutePhase executes all handlers for a specific phase.
// Note: The context should be set via SetContext before starting the ticker.
func (s *Scheduler[T]) ExecutePhase(phase Phase) {
	s.mu.RLock()
	handlers := s.handlers[phase]
	ctx := s.context
	s.mu.RUnlock()

	s.currentPhase.Store(int32(phase))

	for _, handler := range handlers {
		handler(ctx)
	}
}

// ExecuteAllPhases executes all phases in order from PhaseStart to PhaseEnd.
func (s *Scheduler[T]) ExecuteAllPhases() {
	for phase := PhaseStart; phase <= PhaseEnd; phase++ {
		s.ExecutePhase(phase)
	}
}

// CurrentPhase returns the phase currently being executed.
func (s *Scheduler[T]) CurrentPhase() Phase {
	return Phase(s.currentPhase.Load())
}

// HandlerCount returns the number of handlers registered for a phase.
func (s *Scheduler[T]) HandlerCount(phase Phase) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlers[phase])
}
