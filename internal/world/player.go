package world

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

type Entity struct {
	// Data tags
	Motion    [3]float64 // 0: dX, 1: dY, 2: dZ
	NoGravity bool
	OnGround  bool
	Pos       [3]float64 // 0: x, 1: y, 2: z
	Rot       [2]float32 // 0: yaw, 1: pitch
	UUID      uuid.UUID

	// State
	EntityID int32
	Flags    byte
	Pose     int32
}

type LivingEntity struct {
	*Entity
	CanPickupLoot bool
	Health        float32
	LeftHanded    bool
	NoAI          bool
}

type Player struct {
	// Data tags
	*LivingEntity
	foodExhaustionLevel float32
	foodLevel           int32
	foodSaturationLevel float32
	foodTickTimer       int32
	GameMode            uint8 // ntb playerGameType
	PreviousGameMode    int8  // ntb previousPlayerGameType
	PushingAgainstWall  bool  // not sure if this should be there ?
	SelectedItemSlot    int32

	// State
	Inventory         *mc.PlayerInventory // todo: this should maybe be a generic struct, as entities can have inventories too
	World             *World
	ProfileProperties []mc.ProfileProperty
	Name              mc.String
	Loaded            bool
	Input             mc.UnsignedByte
	Movement          MovementTracker
}

type MovementTracker struct {
	PacketCount int
	LastTickX   float64
	LastTickY   float64
	LastTickZ   float64
}

func NewPlayer(uuid uuid.UUID, name mc.String, profileProperties []mc.ProfileProperty, world *World, properties *systems.Properties) *Player {
	player := &Player{
		LivingEntity: &LivingEntity{
			Entity: &Entity{
				EntityID: world.GetNextEntityID(),
				UUID:     uuid,
				OnGround: true,
				Pos:      [3]float64{0, 80, 0},
			},
		},
		World:            world,
		Inventory:        mc.NewPlayerInventory(),
		Name:             name,
		Loaded:           false,
		GameMode:         uint8(properties.GameMode), // handle force-gamemode
		PreviousGameMode: -1,
		Movement: MovementTracker{
			LastTickY: 80,
		},
		ProfileProperties: profileProperties,
	}
	player.Movement.LastTickY = player.Pos[1]

	return player
}

func (player *Player) HasInput(input mc.PlayerInput) bool {
	return player.Input&input != 0
}
