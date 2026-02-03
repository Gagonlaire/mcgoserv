package tick

import (
	"sync"
)

// PhaseHandler is a function that handles a specific phase of the tick.
type PhaseHandler func()

// Scheduler manages the execution order of tick phases and their handlers.
type Scheduler struct {
	// handlers maps phases to their handler functions.
	handlers [PhaseCount][]PhaseHandler

	// mu protects concurrent access to handlers.
	mu sync.RWMutex

	// currentPhase tracks the phase currently being executed.
	currentPhase Phase
}

// NewScheduler creates a new tick scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		handlers:     [PhaseCount][]PhaseHandler{},
		currentPhase: PhaseStart,
	}
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

	s.currentPhase = phase

	for _, handler := range handlers {
		handler()
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
	return s.currentPhase
}

// HandlerCount returns the number of handlers registered for a phase.
func (s *Scheduler) HandlerCount(phase Phase) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.handlers[phase])
}
