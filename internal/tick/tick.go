// Package tick provides the base architecture for Minecraft's game tick system.
// A game tick in Minecraft runs 20 times per second (every 50ms) and is the
// fundamental unit of time for all game mechanics.
//
// Reference: https://minecraft.wiki/w/Tick
package tick

import (
	"time"
)

const (
	// TicksPerSecond is the target tick rate for Minecraft servers (20 TPS).
	TicksPerSecond = 20

	// TickDuration is the target duration for each tick (50ms).
	TickDuration = time.Second / TicksPerSecond

	// TicksPerDay is the number of ticks in a Minecraft day (24000 ticks).
	TicksPerDay = 24000
)

// Phase represents a distinct phase within a game tick.
// The game tick is divided into multiple phases that execute in a specific order.
type Phase int

const (
	// PhaseStart marks the beginning of a tick.
	PhaseStart Phase = iota

	// PhaseBlockEvents processes block events queued from the previous tick.
	PhaseBlockEvents

	// PhaseEntities updates all entities in the world.
	PhaseEntities

	// PhaseBlockEntities updates all block entities (tile entities).
	PhaseBlockEntities

	// PhaseChunks handles chunk-related updates including random ticks and scheduled block ticks.
	PhaseChunks

	// PhaseVillages processes village mechanics (gossip, raids, etc.).
	PhaseVillages

	// PhaseRaids updates active raids.
	PhaseRaids

	// PhaseWeatherAndTime updates weather, time of day, and sleeping mechanics.
	PhaseWeatherAndTime

	// PhaseWanderingTrader handles wandering trader spawning.
	PhaseWanderingTrader

	// PhaseCommands processes scheduled commands and game rules.
	PhaseCommands

	// PhaseWorldBorder updates the world border.
	PhaseWorldBorder

	// PhaseNetwork handles network packet processing.
	PhaseNetwork

	// PhaseEnd marks the end of a tick.
	PhaseEnd

	// PhaseCount is the total number of phases.
	PhaseCount
)

// String returns the name of the phase.
func (p Phase) String() string {
	switch p {
	case PhaseStart:
		return "Start"
	case PhaseBlockEvents:
		return "BlockEvents"
	case PhaseEntities:
		return "Entities"
	case PhaseBlockEntities:
		return "BlockEntities"
	case PhaseChunks:
		return "Chunks"
	case PhaseVillages:
		return "Villages"
	case PhaseRaids:
		return "Raids"
	case PhaseWeatherAndTime:
		return "WeatherAndTime"
	case PhaseWanderingTrader:
		return "WanderingTrader"
	case PhaseCommands:
		return "Commands"
	case PhaseWorldBorder:
		return "WorldBorder"
	case PhaseNetwork:
		return "Network"
	case PhaseEnd:
		return "End"
	default:
		return "Unknown"
	}
}

// GameTime represents the current game time in ticks.
// It tracks both the total tick count and the time of day.
type GameTime struct {
	// TotalTicks is the total number of ticks since the world was created.
	TotalTicks int64

	// DayTime is the current time of day in ticks (0-23999).
	// 0 = sunrise (6:00 AM)
	// 6000 = noon (12:00 PM)
	// 12000 = sunset (6:00 PM)
	// 18000 = midnight (12:00 AM)
	DayTime int64

	// Day is the current day number.
	Day int64
}

// NewGameTime creates a new GameTime starting at dawn of day 1.
func NewGameTime() *GameTime {
	return &GameTime{
		TotalTicks: 0,
		DayTime:    0,
		Day:        1,
	}
}

// Advance moves the game time forward by one tick.
func (gt *GameTime) Advance() {
	gt.TotalTicks++
	gt.DayTime = (gt.DayTime + 1) % TicksPerDay
	if gt.DayTime == 0 {
		gt.Day++
	}
}

// SetDayTime sets the time of day without affecting total ticks.
func (gt *GameTime) SetDayTime(dayTime int64) {
	gt.DayTime = dayTime % TicksPerDay
}

// IsDay returns true if it's currently daytime (between dawn and dusk).
func (gt *GameTime) IsDay() bool {
	return gt.DayTime >= 0 && gt.DayTime < 12000
}

// IsNight returns true if it's currently nighttime.
func (gt *GameTime) IsNight() bool {
	return !gt.IsDay()
}

// TickMetrics holds performance metrics for ticks.
type TickMetrics struct {
	// LastTickDuration is the duration of the last tick.
	LastTickDuration time.Duration

	// AverageTickDuration is the rolling average tick duration.
	AverageTickDuration time.Duration

	// MaxTickDuration is the maximum tick duration recorded.
	MaxTickDuration time.Duration

	// TicksBehind tracks how many ticks the server is lagging behind.
	TicksBehind int64

	// SampleCount is the number of samples in the rolling average.
	SampleCount int64
}

// NewTickMetrics creates a new TickMetrics instance.
func NewTickMetrics() *TickMetrics {
	return &TickMetrics{}
}

// Record records a tick duration and updates metrics.
func (m *TickMetrics) Record(duration time.Duration) {
	m.LastTickDuration = duration
	m.SampleCount++

	// Update rolling average
	if m.SampleCount == 1 {
		m.AverageTickDuration = duration
	} else {
		// Exponential moving average with alpha = 0.1
		alpha := 0.1
		m.AverageTickDuration = time.Duration(
			float64(m.AverageTickDuration)*(1-alpha) + float64(duration)*alpha,
		)
	}

	// Update max
	if duration > m.MaxTickDuration {
		m.MaxTickDuration = duration
	}
}

// TPS returns the current ticks per second based on average tick duration.
func (m *TickMetrics) TPS() float64 {
	if m.AverageTickDuration == 0 {
		return TicksPerSecond
	}
	actualTPS := float64(time.Second) / float64(m.AverageTickDuration)
	if actualTPS > TicksPerSecond {
		return TicksPerSecond
	}
	return actualTPS
}
