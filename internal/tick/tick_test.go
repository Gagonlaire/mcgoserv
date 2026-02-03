package tick

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestPhaseString(t *testing.T) {
	tests := []struct {
		phase    Phase
		expected string
	}{
		{PhaseStart, "Start"},
		{PhaseBlockEvents, "BlockEvents"},
		{PhaseEntities, "Entities"},
		{PhaseBlockEntities, "BlockEntities"},
		{PhaseChunks, "Chunks"},
		{PhaseVillages, "Villages"},
		{PhaseRaids, "Raids"},
		{PhaseWeatherAndTime, "WeatherAndTime"},
		{PhaseWanderingTrader, "WanderingTrader"},
		{PhaseCommands, "Commands"},
		{PhaseWorldBorder, "WorldBorder"},
		{PhaseNetwork, "Network"},
		{PhaseEnd, "End"},
		{Phase(100), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.phase.String(); got != tt.expected {
				t.Errorf("Phase.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGameTime(t *testing.T) {
	t.Run("NewGameTime", func(t *testing.T) {
		gt := NewGameTime()
		if gt.TotalTicks != 0 {
			t.Errorf("TotalTicks = %d, want 0", gt.TotalTicks)
		}
		if gt.DayTime != 0 {
			t.Errorf("DayTime = %d, want 0", gt.DayTime)
		}
		if gt.Day != 1 {
			t.Errorf("Day = %d, want 1", gt.Day)
		}
	})

	t.Run("Advance", func(t *testing.T) {
		gt := NewGameTime()
		gt.Advance()
		if gt.TotalTicks != 1 {
			t.Errorf("TotalTicks = %d, want 1", gt.TotalTicks)
		}
		if gt.DayTime != 1 {
			t.Errorf("DayTime = %d, want 1", gt.DayTime)
		}
	})

	t.Run("DayRollover", func(t *testing.T) {
		gt := NewGameTime()
		gt.DayTime = TicksPerDay - 1
		gt.Advance()
		if gt.DayTime != 0 {
			t.Errorf("DayTime = %d, want 0", gt.DayTime)
		}
		if gt.Day != 2 {
			t.Errorf("Day = %d, want 2", gt.Day)
		}
	})

	t.Run("SetDayTime", func(t *testing.T) {
		gt := NewGameTime()
		gt.SetDayTime(12000)
		if gt.DayTime != 12000 {
			t.Errorf("DayTime = %d, want 12000", gt.DayTime)
		}
	})

	t.Run("SetDayTimeWraps", func(t *testing.T) {
		gt := NewGameTime()
		gt.SetDayTime(TicksPerDay + 1000)
		if gt.DayTime != 1000 {
			t.Errorf("DayTime = %d, want 1000", gt.DayTime)
		}
	})

	t.Run("IsDay", func(t *testing.T) {
		gt := NewGameTime()
		gt.DayTime = 6000 // Noon
		if !gt.IsDay() {
			t.Error("expected IsDay() to be true at noon")
		}
	})

	t.Run("IsNight", func(t *testing.T) {
		gt := NewGameTime()
		gt.DayTime = 18000 // Midnight
		if !gt.IsNight() {
			t.Error("expected IsNight() to be true at midnight")
		}
	})
}

func TestTickMetrics(t *testing.T) {
	t.Run("NewTickMetrics", func(t *testing.T) {
		m := NewTickMetrics()
		if m.LastTickDuration != 0 {
			t.Errorf("LastTickDuration = %v, want 0", m.LastTickDuration)
		}
		if m.TPS() != TicksPerSecond {
			t.Errorf("TPS() = %v, want %v", m.TPS(), float64(TicksPerSecond))
		}
	})

	t.Run("Record", func(t *testing.T) {
		m := NewTickMetrics()
		m.Record(40 * time.Millisecond)
		if m.LastTickDuration != 40*time.Millisecond {
			t.Errorf("LastTickDuration = %v, want 40ms", m.LastTickDuration)
		}
		if m.SampleCount != 1 {
			t.Errorf("SampleCount = %d, want 1", m.SampleCount)
		}
	})

	t.Run("MaxTickDuration", func(t *testing.T) {
		m := NewTickMetrics()
		m.Record(30 * time.Millisecond)
		m.Record(50 * time.Millisecond)
		m.Record(40 * time.Millisecond)
		if m.MaxTickDuration != 50*time.Millisecond {
			t.Errorf("MaxTickDuration = %v, want 50ms", m.MaxTickDuration)
		}
	})

	t.Run("TPS_MaxBounded", func(t *testing.T) {
		m := NewTickMetrics()
		m.Record(10 * time.Millisecond) // Very fast tick
		// TPS should be capped at TicksPerSecond
		if m.TPS() != TicksPerSecond {
			t.Errorf("TPS() = %v, want %v", m.TPS(), float64(TicksPerSecond))
		}
	})
}

func TestScheduler(t *testing.T) {
	t.Run("NewScheduler", func(t *testing.T) {
		s := NewScheduler()
		if s == nil {
			t.Fatal("NewScheduler() returned nil")
		}
		if s.CurrentPhase() != PhaseStart {
			t.Errorf("CurrentPhase() = %v, want %v", s.CurrentPhase(), PhaseStart)
		}
	})

	t.Run("RegisterHandler", func(t *testing.T) {
		s := NewScheduler()
		var called bool
		s.RegisterHandler(PhaseEntities, func() { called = true })

		if s.HandlerCount(PhaseEntities) != 1 {
			t.Errorf("HandlerCount = %d, want 1", s.HandlerCount(PhaseEntities))
		}

		s.ExecutePhase(PhaseEntities)
		if !called {
			t.Error("handler was not called")
		}
	})

	t.Run("MultipleHandlers", func(t *testing.T) {
		s := NewScheduler()
		var order []int
		s.RegisterHandler(PhaseEntities, func() { order = append(order, 1) })
		s.RegisterHandler(PhaseEntities, func() { order = append(order, 2) })
		s.RegisterHandler(PhaseEntities, func() { order = append(order, 3) })

		s.ExecutePhase(PhaseEntities)

		if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
			t.Errorf("handlers executed in wrong order: %v", order)
		}
	})

	t.Run("UnregisterAllHandlers", func(t *testing.T) {
		s := NewScheduler()
		s.RegisterHandler(PhaseEntities, func() {})
		s.RegisterHandler(PhaseEntities, func() {})
		s.UnregisterAllHandlers(PhaseEntities)

		if s.HandlerCount(PhaseEntities) != 0 {
			t.Errorf("HandlerCount = %d, want 0", s.HandlerCount(PhaseEntities))
		}
	})

	t.Run("ExecuteAllPhases", func(t *testing.T) {
		s := NewScheduler()
		var phases []Phase
		for p := PhaseStart; p <= PhaseEnd; p++ {
			phase := p // capture
			s.RegisterHandler(p, func() { phases = append(phases, phase) })
		}

		s.ExecuteAllPhases()

		if len(phases) != int(PhaseEnd)+1 {
			t.Errorf("expected %d phases to execute, got %d", int(PhaseEnd)+1, len(phases))
		}

		// Verify order
		for i, p := range phases {
			if p != Phase(i) {
				t.Errorf("phase %d executed at position %d, expected %d", p, i, i)
			}
		}
	})
}

func TestTicker(t *testing.T) {
	t.Run("NewTicker", func(t *testing.T) {
		ticker := NewTicker()
		if ticker == nil {
			t.Fatal("NewTicker() returned nil")
		}
		if ticker.Scheduler() == nil {
			t.Error("Scheduler() returned nil")
		}
		// GameTime and Metrics now return values (snapshots), not pointers
		gt := ticker.GameTime()
		if gt.TotalTicks < 0 {
			t.Error("GameTime() returned invalid state")
		}
		m := ticker.Metrics()
		if m.SampleCount < 0 {
			t.Error("Metrics() returned invalid state")
		}
	})

	t.Run("StartStop", func(t *testing.T) {
		ticker := NewTicker()

		go ticker.Start()
		time.Sleep(10 * time.Millisecond)

		if !ticker.IsRunning() {
			t.Error("expected ticker to be running")
		}

		ticker.Stop()

		if ticker.IsRunning() {
			t.Error("expected ticker to be stopped")
		}
	})

	t.Run("TickCallback", func(t *testing.T) {
		ticker := NewTicker()
		var tickCount int64

		ticker.SetOnTick(func(tickNumber int64) {
			atomic.AddInt64(&tickCount, 1)
		})

		go ticker.Start()
		time.Sleep(150 * time.Millisecond) // Allow ~3 ticks
		ticker.Stop()

		count := atomic.LoadInt64(&tickCount)
		if count < 2 {
			t.Errorf("expected at least 2 ticks, got %d", count)
		}
	})

	t.Run("GameTimeAdvances", func(t *testing.T) {
		ticker := NewTicker()

		go ticker.Start()
		time.Sleep(150 * time.Millisecond) // Allow ~3 ticks
		ticker.Stop()

		if ticker.CurrentTick() < 2 {
			t.Errorf("expected at least 2 ticks, got %d", ticker.CurrentTick())
		}
	})

	t.Run("PhaseHandlersExecute", func(t *testing.T) {
		ticker := NewTicker()
		var entitiesCalled int64
		var chunksCalled int64

		ticker.Scheduler().RegisterHandler(PhaseEntities, func() {
			atomic.AddInt64(&entitiesCalled, 1)
		})
		ticker.Scheduler().RegisterHandler(PhaseChunks, func() {
			atomic.AddInt64(&chunksCalled, 1)
		})

		go ticker.Start()
		time.Sleep(150 * time.Millisecond) // Allow ~3 ticks
		ticker.Stop()

		if atomic.LoadInt64(&entitiesCalled) < 2 {
			t.Errorf("expected entities handler to be called at least 2 times, got %d", entitiesCalled)
		}
		if atomic.LoadInt64(&chunksCalled) < 2 {
			t.Errorf("expected chunks handler to be called at least 2 times, got %d", chunksCalled)
		}
	})
}

func TestConstants(t *testing.T) {
	t.Run("TicksPerSecond", func(t *testing.T) {
		if TicksPerSecond != 20 {
			t.Errorf("TicksPerSecond = %d, want 20", TicksPerSecond)
		}
	})

	t.Run("TickDuration", func(t *testing.T) {
		if TickDuration != 50*time.Millisecond {
			t.Errorf("TickDuration = %v, want 50ms", TickDuration)
		}
	})

	t.Run("TicksPerDay", func(t *testing.T) {
		if TicksPerDay != 24000 {
			t.Errorf("TicksPerDay = %d, want 24000", TicksPerDay)
		}
	})
}
