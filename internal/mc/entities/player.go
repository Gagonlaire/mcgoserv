package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/layers"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

const (
	IndexAdditionalHearts       metadata.Index = 17
	IndexScore                  metadata.Index = 18
	IndexLeftShoulderEntryData  metadata.Index = 19
	IndexRightShoulderEntryData metadata.Index = 20
)

//meta:encode mode=entity parents=LivingEntity,AvatarData nbt=getters
type Player struct {
	Leash               struct{}            `nbt:"-"`
	DropChances         struct{}            `nbt:"-"`
	CustomName          struct{}            `nbt:"-"`
	CustomNameVisible   struct{}            `nbt:"-"`
	Glowing             struct{}            `nbt:"-"`
	CanPickUpLoot       struct{}            `nbt:"-"`
	LeftHanded          struct{}            `nbt:"-"`
	PersistenceRequired struct{}            `nbt:"-"`
	Inventory           *mc.PlayerInventory `nbt:"-"` // todo: implement inventories (containers)
	layers.AvatarData
	Name              string               `nbt:"-"`
	Dimension         string               // todo: should be a identifier (ex: minecraft:overworld)
	ProfileProperties []mc.ProfileProperty `nbt:"-"`
	EnderItems        []any                // todo: implement inventories (containers)
	Information       mc.ClientInformation `nbt:"-"`
	Movement          MovementTracker      `nbt:"-"`
	LivingEntity
	ChatSession                          mc.ChatSession                             `nbt:"-"`
	CurrentExplosionImpactPos            [3]float64                                 `nbt:"current_explosion_impact_pos"`
	EnteredNetherPos                     [3]float64                                 `nbt:"entered_nether_pos"`
	PermissionLevel                      int                                        `nbt:"-"`
	LeftShoulder                         mc.PrefixedOptional[mc.VarInt, *mc.VarInt] `meta:"IndexLeftShoulderEntryData,OptVarInt"` // todo: create a nbt encode function
	RightShoulder                        mc.PrefixedOptional[mc.VarInt, *mc.VarInt] `meta:"IndexRightShoulderEntryData,OptVarInt"`
	FoodExhaustionLevel                  float32                                    `nbt:"foodExhaustionLevel"`
	XpP                                  float32
	FoodLevel                            int32 `nbt:"foodLevel"`
	FoodTickTimer                        int32 `nbt:"foodTickTimer"`
	SelectedItemSlot                     int32
	DataVersion                          int32
	XpLevel                              int32
	XpSeed                               int32
	XpTotal                              int32
	IgnoreFallDamageFromCurrentExplosion bool    `nbt:"ignore_fall_damage_from_current_explosion"`
	FoodSaturationLevel                  float32 `nbt:"foodSaturationLevel"`
	Score                                int32   `meta:"IndexScore,VarInt"`
	AdditionalHearts                     float32 `meta:"IndexAdditionalHearts,Float" nbt:"-"`
	PushingAgainstWall                   bool    `nbt:"-"`
	PreviousGameMode                     int32   `nbt:"previousPlayerGameType"`
	GameMode                             int32   `nbt:"playerGameType"`
	Loaded                               bool    `nbt:"-"`
	Input                                byte    `nbt:"-"`
	SleepTimer                           int16
	SeenCredits                          bool `nbt:"seenCredits"`
	// todo: implement LastDeathLocation, abilities, respawn, warden_spawn_tracker, recipeBook, RootVehicle, ShoulderEntityLeft, ShoulderEntityRight
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
				Position: [3]float64{0, 80, 0},
				UUID:     NbtUUID(UUID),
				OnGround: true,
				TypeID:   mcdata.EntityPlayer,
			},
			Health: 20.0,
		},
		Inventory:         mc.NewPlayerInventory(),
		Name:              name,
		Loaded:            false,
		PermissionLevel:   permissionLevel,
		GameMode:          int32(cfg.Server.GameMode),
		PreviousGameMode:  -1,
		ProfileProperties: profileProperties,
	}
	player.Movement.LastTickY = player.Position[1]
	player.Movement.VisibleChunks = make(map[mc.ChunkPos]struct{})
	player.Information.ViewDistance = mc.Byte(cfg.Performance.MaxViewDistance)
	player.Information.AllowServerListings = true
	player.ChatSession.Signed = false

	player.AvatarData.Init(player)
	player.InitDefaults()

	return player
}

func (p *Player) Tick() {}
