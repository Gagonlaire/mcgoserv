package world

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type World struct {
	Dimensions     map[string]*Dimension
	Rules          GameRules
	Time           int64
	DayTime        int64
	Day            int64
	NextTimeUpdate int64
	Players        map[mc.UUID]*Player
	lastEntityID   mc.Int
}

type GameRules struct {
	DoDaylightCycle bool
}

type GenSettings struct {
	Seed int64
}

func NewWorld() *World {
	world := &World{
		Dimensions: map[string]*Dimension{
			"minecraft:overworld": {
				Type: DefaultDimensionsType["minecraft:overworld"],
			},
			"minecraft:the_nether": {
				Type: DefaultDimensionsType["minecraft:the_nether"],
			},
			"minecraft:the_end": {
				Type: DefaultDimensionsType["minecraft:the_end"],
			},
		},
		Players: make(map[mc.UUID]*Player),
	}

	for _, v := range world.Dimensions {
		v.World = world
	}

	return world
}

func (w *World) GetNextEntityID() mc.Int {
	// todo: replace with a real id distribution
	w.lastEntityID++
	return w.lastEntityID
}

func (w *World) AddPlayer(p *Player) {
	w.Players[p.UUID] = p
}

func (w *World) RemovePlayer(uuid mc.UUID) {
	delete(w.Players, uuid)
}
