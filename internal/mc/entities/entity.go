package entities

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

type BaseEntity struct {
	metadata.DirtyTracker
	DimensionID string // todo: change to a numeric id
	// todo: fix custom, mc.PrefixedOptional[tc.Component, *tc.Component] doesn't work because of interface
	CustomName        string
	CustomNameVisible bool
	Motion            [3]float64
	Pos               [3]float64
	TypeID            mcdata.EntityType
	Rot               [2]float32
	FallDistance      float32
	EntityID          int32
	Fire              int16
	Air               int16
	UUID              uuid.UUID
	Pose              EntityPose
	Flags             EntityFlag
	OnGround          bool
	NoGravity         bool
	Silent            bool
	InSyncQueue       bool
	TickFrozen        int32
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

func (e *BaseEntity) SetFlags(flags EntityFlag) {
	if e.Flags != flags {
		e.Flags = flags
		e.MarkDirty(IndexEntityFlags)
	}
}

func (e *BaseEntity) SetFlag(flag EntityFlag, on bool) {
	var newFlags EntityFlag
	if on {
		newFlags = e.Flags | flag
	} else {
		newFlags = e.Flags &^ flag
	}
	e.SetFlags(newFlags)
}

func (e *BaseEntity) SetAir(air int16) {
	if e.Air != air {
		e.Air = air
		e.MarkDirty(IndexAirTicks)
	}
}

func (e *BaseEntity) SetSilent(silent bool) {
	if e.Silent != silent {
		e.Silent = silent
		e.MarkDirty(IndexSilent)
	}
}

func (e *BaseEntity) SetNoGravity(noGravity bool) {
	if e.NoGravity != noGravity {
		e.NoGravity = noGravity
		e.MarkDirty(IndexNoGravity)
	}
}

func (e *BaseEntity) SetPose(pose EntityPose) {
	if e.Pose != pose {
		e.Pose = pose
		e.MarkDirty(IndexPose)
	}
}

func (e *BaseEntity) EncodeMetadata(pkt *packet.OutboundPacket) {
	if e.DirtyTracker.IsDirty(IndexEntityFlags) {
		_ = pkt.Encode(mc.UnsignedByte(IndexEntityFlags), mc.VarInt(metadata.TypeByte), mc.Byte(e.Flags))
	}
	if e.DirtyTracker.IsDirty(IndexAirTicks) {
		_ = pkt.Encode(mc.UnsignedByte(IndexAirTicks), mc.VarInt(metadata.TypeVarInt), mc.VarInt(e.Air))
	}
	if e.DirtyTracker.IsDirty(IndexCustomNameVisible) {
		_ = pkt.Encode(mc.UnsignedByte(IndexCustomNameVisible), mc.VarInt(metadata.TypeBoolean), mc.Boolean(e.CustomNameVisible))
	}
	if e.DirtyTracker.IsDirty(IndexSilent) {
		_ = pkt.Encode(mc.UnsignedByte(IndexSilent), mc.VarInt(metadata.TypeBoolean), mc.Boolean(e.Silent))
	}
	if e.DirtyTracker.IsDirty(IndexNoGravity) {
		_ = pkt.Encode(mc.UnsignedByte(IndexNoGravity), mc.VarInt(metadata.TypeBoolean), mc.Boolean(e.NoGravity))
	}
	if e.DirtyTracker.IsDirty(IndexPose) {
		_ = pkt.Encode(mc.UnsignedByte(IndexPose), mc.VarInt(metadata.TypePose), mc.VarInt(e.Pose))
	}
	if e.DirtyTracker.IsDirty(IndexTicksFrozen) {
		_ = pkt.Encode(mc.UnsignedByte(IndexTicksFrozen), mc.VarInt(metadata.TypeVarInt), mc.VarInt(e.TickFrozen))
	}
}
