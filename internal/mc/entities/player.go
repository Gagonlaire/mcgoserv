package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

type Player struct {
	LivingEntity
	Inventory           *mc.PlayerInventory
	Name                string
	ProfileProperties   []mc.ProfileProperty
	Information         mc.ClientInformation
	Movement            MovementTracker
	ChatSession         mc.ChatSession
	PermissionLevel     int
	SelectedItemSlot    int32
	FoodTickTimer       int32
	FoodSaturationLevel float32
	FoodLevel           int32
	FoodExhaustionLevel float32
	Score               int32
	PushingAgainstWall  bool
	PreviousGameMode    int8
	GameMode            uint8
	Loaded              bool
	Input               byte
}

type MovementTracker struct {
	VisibleChunks   map[mc.ChunkPos]struct{}
	KeepChunks      map[mc.ChunkPos]bool
	PacketCount     int
	PendingTeleport int32
	LastTickX       float64
	LastTickY       float64
	LastTickZ       float64
	LastChunkX      int
	LastChunkZ      int
}

func NewPlayer(
	UUID uuid.UUID,
	name string,
	permissionLevel int,
	profileProperties []mc.ProfileProperty,
	cfg *systems.Config,
) *Player {
	player := &Player{
		LivingEntity: LivingEntity{
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
		GameMode:          uint8(cfg.Server.GameMode), // todo: handle force-gamemode
		PreviousGameMode:  -1,
		ProfileProperties: profileProperties,
	}
	player.Movement.LastTickY = player.Pos[1]
	player.Movement.VisibleChunks = make(map[mc.ChunkPos]struct{})
	player.Information.ViewDistance = mc.Byte(cfg.Performance.MaxViewDistance)
	player.Information.AllowServerListings = true
	player.ChatSession.Signed = false

	return player
}

func (p *Player) Tick() {}
