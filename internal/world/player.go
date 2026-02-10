package world

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type Player struct {
	EntityID   mc.Int
	UUID       mc.UUID
	Name       string
	World      *World
	X, Y, Z    mc.Double
	Yaw, Pitch float32
}

func NewPlayer(id mc.Int, uuid mc.UUID, name string, w *World) *Player {
	return &Player{
		EntityID: id,
		UUID:     uuid,
		Name:     name,
		World:    w,
		X:        0,
		Y:        80,
		Z:        0,
		Yaw:      0,
		Pitch:    0,
	}
}
