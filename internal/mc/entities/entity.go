package entities

//go:generate go run ../../../cmd/gen-meta .

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/google/uuid"
)

type Entity interface {
	EncodeMetadata(pkt *packet.OutboundPacket)
	HasMetaChanges() bool
	ClearMetaChanges()
	GetID() int32
	GetUUID() uuid.UUID
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

//meta:encode mode=entity
type BaseEntity struct {
	metadata.DirtyTracker
	DimensionID       string              // todo: change to a numeric id
	CustomName        mc.OptTextComponent `meta:"IndexCustomName,OptTextComponent"`
	CustomNameVisible bool                `meta:"IndexCustomNameVisible,Boolean"`
	Motion            [3]float64
	Pos               [3]float64
	TypeID            mcdata.EntityType
	Rot               [2]float32
	FallDistance      float32
	EntityID          int32
	Fire              int16
	Air               int32 `meta:"IndexAirTicks,VarInt,default=300"`
	UUID              uuid.UUID
	Pose              EntityPose `meta:"IndexPose,Pose,default=EntityPoseStanding"`
	Flags             EntityFlag `meta:"IndexEntityFlags,Byte,flags"`
	OnGround          bool
	NoGravity         bool `meta:"IndexNoGravity,Boolean"`
	Silent            bool `meta:"IndexSilent,Boolean"`
	InSyncQueue       bool
	TickFrozen        int32 `meta:"IndexTicksFrozen,VarInt"`
}

func (e *BaseEntity) GetID() int32               { return e.EntityID }
func (e *BaseEntity) GetUUID() uuid.UUID         { return e.UUID }
func (e *BaseEntity) GetType() mcdata.EntityType { return e.TypeID }
func (e *BaseEntity) GetPos() [3]float64         { return e.Pos }
func (e *BaseEntity) SetPos(pos [3]float64)      { e.Pos = pos }
func (e *BaseEntity) GetRot() [2]float32         { return e.Rot }
func (e *BaseEntity) SetRot(rot [2]float32)      { e.Rot = rot }
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
