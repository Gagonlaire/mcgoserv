package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

type Player struct {
	*LivingEntity
	Inventory           *mc.PlayerInventory
	Name                string
	ProfileProperties   []mc.ProfileProperty
	Information         mc.ClientInformation
	Movement            MovementTracker
	ChatSession         mc.ChatSession
	PermissionLevel     int
	SelectedItemSlot    int32
	FoodTickTimer       int32   `nbt:"foodTickTimer"`
	FoodSaturationLevel float32 `nbt:"foodSaturationLevel"`
	FoodLevel           int32   `nbt:"foodLevel"`
	FoodExhaustionLevel float32 `nbt:"foodExhaustionLevel"`
	Score               int32
	PushingAgainstWall  bool
	PreviousGameMode    int8
	GameMode            uint8
	Loaded              bool
	Input               byte
}

type MovementTracker struct {
	VisibleChunks map[mc.ChunkPos]struct{}
	KeepChunks    map[mc.ChunkPos]bool
	PacketCount   int
	LastTickX     float64
	LastTickY     float64
	LastTickZ     float64
	LastChunkX    int
	LastChunkZ    int
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
