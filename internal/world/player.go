package world

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type Player struct {
	EntityID   mc.Int
	UUID       mc.UUID
	Name       mc.String
	World      *World
	X, Y, Z    mc.Double
	Yaw, Pitch mc.Float
	onGround   mc.Boolean
	Movement   MovementTracker
}

type MovementTracker struct {
	PacketCount int
	LastTickX   float64
	LastTickY   float64
	LastTickZ   float64
}

func NewPlayer(id mc.Int, uuid mc.UUID, name mc.String, w *World) *Player {
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
		Movement: MovementTracker{
			LastTickY: 80,
		},
	}
}
