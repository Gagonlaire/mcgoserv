package tick

import (
	"sync"
	"sync/atomic"
)

// PhaseHandler is a function that handles a specific phase of the tick.
// It receives a context parameter that can be used to pass server or other state.
type PhaseHandler func(ctx any)

// Scheduler manages the execution order of tick phases and their handlers.
type Scheduler struct {
	// handlers maps phases to their handler functions.
	handlers [PhaseCount][]PhaseHandler

	// mu protects concurrent access to handlers.
	mu sync.RWMutex

	// currentPhase tracks the phase currently being executed (accessed atomically).
	currentPhase atomic.Int32

	// context is passed to all phase handlers during execution (accessed atomically).
	context atomic.Value
}

// NewScheduler creates a new tick scheduler.
func NewScheduler() *Scheduler {
	s := &Scheduler{
		handlers: [PhaseCount][]PhaseHandler{},
	}
	s.currentPhase.Store(int32(PhaseStart))
	return s
}

// SetContext sets the context that will be passed to all phase handlers.
// This is typically called once during initialization with the server instance.
func (s *Scheduler) SetContext(ctx any) {
	s.context.Store(ctx)
}

// Context returns the current context.
func (s *Scheduler) Context() any {
	return s.context.Load()
}

// RegisterHandler registers a handler for a specific phase.
// Multiple handlers can be registered for the same phase and will be
// executed in the order they were registered.
func (s *Scheduler) RegisterHandler(phase Phase, handler PhaseHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[phase] = append(s.handlers[phase], handler)
}

// UnregisterAllHandlers removes all handlers for a specific phase.
func (s *Scheduler) UnregisterAllHandlers(phase Phase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[phase] = nil
}

// ExecutePhase executes all handlers for a specific phase.
func (s *Scheduler) ExecutePhase(phase Phase) {
	s.mu.RLock()
	handlers := s.handlers[phase]
	s.mu.RUnlock()

	ctx := s.context.Load()
	s.currentPhase.Store(int32(phase))

	for _, handler := range handlers {
		handler(ctx)
	}
}

// ExecuteAllPhases executes all phases in order from PhaseStart to PhaseEnd.
func (s *Scheduler) ExecuteAllPhases() {
	for phase := PhaseStart; phase <= PhaseEnd; phase++ {
		s.ExecutePhase(phase)
	}
}

// CurrentPhase returns the phase currently being executed.
func (s *Scheduler) CurrentPhase() Phase {
	return Phase(s.currentPhase.Load())
}

// HandlerCount returns the number of handlers registered for a phase.
func (s *Scheduler) HandlerCount(phase Phase) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlers[phase])
}
