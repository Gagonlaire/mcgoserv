package entities

//go:generate go run ../../../cmd/gen-meta .

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Entity interface {
	EncodeMetadata(pkt *packet.OutboundPacket)
	HasMetaChanges() bool
	ClearMetaChanges()
	GetID() int32
	GetUUID() NbtUUID
	GetType() mcdata.EntityType
	GetPos() [3]float64
	SetPos(pos [3]float64)
	GetRot() [2]float32
	SetRot(rot [2]float32)
	GetMotion() [3]float64
	IsOnGround() bool
	Tick()
	Base() *BaseEntity
}

const (
	IndexEntityFlags       metadata.Index = 0
	IndexAirTicks          metadata.Index = 1
	IndexCustomName        metadata.Index = 2
	IndexCustomNameVisible metadata.Index = 3
	IndexSilent            metadata.Index = 4
	IndexNoGravity         metadata.Index = 5
	IndexPose              metadata.Index = 6
	IndexTicksFrozen       metadata.Index = 7
)

type EntityFlag byte

const (
	EntityFlagNone      EntityFlag = 0
	EntityFlagOnFire    EntityFlag = 0x01
	EntityFlagCrouching EntityFlag = 0x02
	EntityFlagSprinting EntityFlag = 0x08
	EntityFlagSwimming  EntityFlag = 0x10
	EntityFlagInvisible EntityFlag = 0x20
	EntityFlagGlowing   EntityFlag = 0x40
	EntityFlagElytra    EntityFlag = 0x80
)

type EntityPose int32

const (
	EntityPoseStanding EntityPose = iota
	EntityPoseFallFlying
	EntityPoseSleeping
	EntityPoseSwimming
	EntityPoseSpinAttack
	EntityPoseSneaking
	EntityPoseLongJumping
	EntityPoseDying
	EntityPoseCroaking
	EntityPoseUsingTongue
	EntityPoseSitting
	EntityPoseRoaring
	EntityPoseSniffing
	EntityPoseEmerging
	EntityPoseDigging
	EntityPoseSliding
	EntityPoseShooting
	EntityPoseInhaling
)

// BaseEntity todo: we should wrap during load/save entities in a struct that hold the type id, exposed for nbt parser
//
//meta:encode mode=entity nbt=accessors
type BaseEntity struct {
	metadata.DirtyTracker `nbt:"-"`
	DimensionID           string              `nbt:"-"`                                         // todo: change to a numeric id
	CustomName            mc.OptTextComponent `meta:"IndexCustomName,OptTextComponent" nbt:"-"` // todo: add text component ntb encoding
	CustomNameVisible     bool                `meta:"IndexCustomNameVisible,Boolean" nbt:"omitempty"`
	Motion                [3]float64
	Position              [3]float64        `nbt:"Pos"`
	TypeID                mcdata.EntityType `nbt:"-"`
	Rotation              [2]float32
	EntityID              int32 `nbt:"-"`
	Air                   int16 `meta:"IndexAirTicks,VarInt,default=300"`
	Fire                  int16
	UUID                  NbtUUID
	Pose                  EntityPose `meta:"IndexPose,Pose,default=EntityPoseStanding" nbt:"-"`
	Flags                 EntityFlag `meta:"IndexEntityFlags,Byte,flags" nbt:"-"`
	OnGround              bool
	NoGravity             bool    `meta:"IndexNoGravity,Boolean" nbt:"omitempty"`
	Silent                bool    `meta:"IndexSilent,Boolean" nbt:"omitempty"`
	InSyncQueue           bool    `nbt:"-"` // used for metadata sync
	TicksFrozen           int32   `meta:"IndexTicksFrozen,VarInt" nbt:"omitempty"`
	FallDistance          float64 `nbt:"fall_distance"`
	Glowing               bool    `nbt:"omitempty"` // this is an alias for EntityFlagGlowing entity flag
	Invulnerable          bool
	PortalCooldown        int32
	// todo: implement Passengers, Tags and data
}

func (e *BaseEntity) GetID() int32               { return e.EntityID }
func (e *BaseEntity) GetUUID() NbtUUID           { return e.UUID }
func (e *BaseEntity) GetType() mcdata.EntityType { return e.TypeID }
func (e *BaseEntity) GetPos() [3]float64         { return e.Position }
func (e *BaseEntity) SetPos(pos [3]float64)      { e.Position = pos }
func (e *BaseEntity) GetRot() [2]float32         { return e.Rotation }
func (e *BaseEntity) SetRot(rot [2]float32)      { e.Rotation = rot }
func (e *BaseEntity) GetMotion() [3]float64      { return e.Motion }
func (e *BaseEntity) IsOnGround() bool           { return e.OnGround }
func (e *BaseEntity) Tick()                      {}
func (e *BaseEntity) Base() *BaseEntity          { return e }

func (e *BaseEntity) MarkDirty(index byte) {
	e.DirtyTracker.Mark(index)
}

func (e *BaseEntity) HasMetaChanges() bool {
	return e.DirtyTracker.HasChanges()
}

func (e *BaseEntity) ClearMetaChanges() {
	e.DirtyTracker.Clear()
	e.InSyncQueue = false
}
