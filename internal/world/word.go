package world

import (
	"github.com/google/uuid"
)

type World struct {
	Dimensions     map[string]*Dimension
	Rules          GameRules
	Time           int64
	DayTime        int64
	Day            int64
	NextTimeUpdate int64
	Players        map[uuid.UUID]*Player
	lastEntityID   int32
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
		Players: make(map[uuid.UUID]*Player),
	}

	for _, v := range world.Dimensions {
		v.World = world
	}

	return world
}

func (w *World) GetNextEntityID() int32 {
	// todo: replace with a real id distribution
	w.lastEntityID++
	return w.lastEntityID
}

func (w *World) AddPlayer(p *Player) {
	w.Players[p.UUID] = p
}

func (w *World) RemovePlayer(uuid uuid.UUID) {
	delete(w.Players, uuid)
}
