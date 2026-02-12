package world

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type Player struct {
	World            *World
	EntityID         mc.VarInt
	UUID             mc.UUID
	Name             mc.String
	Loaded           bool
	IsSneaking       bool
	IsSprinting      bool
	GameMode         mc.UnsignedByte
	PreviousGameMode mc.Byte
	Input            mc.UnsignedByte
	Pose             mc.Pose
	Position         struct {
		X, Y, Z    mc.Double
		Yaw, Pitch mc.Float
		Flags      mc.Byte // 0x01 on ground, 0x02 pushing against a wall
	}
	Movement MovementTracker
}

type MovementTracker struct {
	PacketCount int
	LastTickX   float64
	LastTickY   float64
	LastTickZ   float64
}

func NewPlayer(id mc.VarInt, uuid mc.UUID, name mc.String, w *World) *Player {
	// todo: get current gamemode
	player := &Player{
		World:    w,
		EntityID: id,
		UUID:     uuid,
		Name:     name,
		Loaded:   false,
		Movement: MovementTracker{
			LastTickY: 80,
		},
	}
	player.Position.Y = 80
	player.Position.Flags = 1

	return player
}

func (player *Player) HasInput(input mc.PlayerInput) bool {
	return player.Input&input != 0
}
