package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

type Player struct {
	// Data tags
	*LivingEntity
	GameMode            uint8
	PreviousGameMode    int8
	PushingAgainstWall  bool
	SelectedItemSlot    int32
	Score               int32
	FoodExhaustionLevel float32 `nbt:"foodExhaustionLevel"`
	FoodLevel           int32   `nbt:"foodLevel"`
	FoodSaturationLevel float32 `nbt:"foodSaturationLevel"`
	FoodTickTimer       int32   `nbt:"foodTickTimer"`

	// State
	Inventory         *mc.PlayerInventory
	Movement          MovementTracker
	ProfileProperties []mc.ProfileProperty
	Name              mc.String
	Loaded            bool
	Input             mc.UnsignedByte
}

type MovementTracker struct {
	PacketCount int
	LastTickX   float64
	LastTickY   float64
	LastTickZ   float64
}

var lastEntityID int32 = 1

func NewPlayer(UUID uuid.UUID, name string, profileProperties []mc.ProfileProperty, properties *systems.Properties) *Player {
	player := &Player{
		LivingEntity: &LivingEntity{
			BaseEntity: BaseEntity{
				EntityID: lastEntityID,
				Pos:      [3]float64{0, 80, 0},
				UUID:     UUID,
				OnGround: true,
			},
			Health: 20.0,
		},
		Inventory:         mc.NewPlayerInventory(),
		Name:              mc.String(name),
		Loaded:            false,
		GameMode:          uint8(properties.GameMode), // todo: handle force-gamemode
		PreviousGameMode:  -1,
		ProfileProperties: profileProperties,
	}
	player.Movement.LastTickY = player.Pos[1]

	lastEntityID++

	return player
}

func (p *Player) Tick() {}
