package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
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
	ChatSession       mc.ChatSession
	Inventory         *mc.PlayerInventory
	Movement          MovementTracker
	Information       mc.PlayerInformation
	ProfileProperties []mc.ProfileProperty
	PermissionLevel   int
	Name              string
	Loaded            bool
	Input             byte
}

type MovementTracker struct {
	PacketCount   int
	LastTickX     float64
	LastTickY     float64
	LastTickZ     float64
	LastChunkX    int
	LastChunkZ    int
	VisibleChunks map[mc.ChunkPos]struct{}
}

func NewPlayer(
	UUID uuid.UUID,
	name string,
	permissionLevel int,
	profileProperties []mc.ProfileProperty,
	properties *systems.Properties,
) *Player {
	player := &Player{
		LivingEntity: &LivingEntity{
			BaseEntity: BaseEntity{
				EntityID: 0,
				Pos:      [3]float64{0, 80, 0},
				UUID:     UUID,
				OnGround: true,
				TypeID:   mcdata.EntityPlayer,
			},
			Health: 20.0,
		},
		Inventory:         mc.NewPlayerInventory(),
		Name:              name,
		Loaded:            false,
		PermissionLevel:   permissionLevel,
		GameMode:          uint8(properties.GameMode), // todo: handle force-gamemode
		PreviousGameMode:  -1,
		ProfileProperties: profileProperties,
	}
	player.Movement.LastTickY = player.Pos[1]
	player.Movement.VisibleChunks = make(map[mc.ChunkPos]struct{})
	player.Information.ViewDistance = mc.Byte(properties.ViewDistance)
	player.Information.AllowServerListings = true
	player.ChatSession.Signed = false

	return player
}

func (p *Player) Tick() {}
