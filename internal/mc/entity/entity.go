package entity

//go:generate go run ../../../cmd/gen-meta .

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type Entity interface {
	EncodeMetadata(pkt *packet.OutboundPacket)
	HasMetaChanges() bool
	ClearMetaChanges()
	GetID() int32
	GetUUID() NbtUUID
	GetType() ID
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

type Flag byte

const (
	FlagNone      Flag = 0
	FlagOnFire    Flag = 0x01
	FlagCrouching Flag = 0x02
	FlagSprinting Flag = 0x08
	FlagSwimming  Flag = 0x10
	FlagInvisible Flag = 0x20
	FlagGlowing   Flag = 0x40
	FlagElytra    Flag = 0x80
)

type Pose int32

const (
	PoseStanding Pose = iota
	PoseFallFlying
	PoseSleeping
	PoseSwimming
	PoseSpinAttack
	PoseSneaking
	PoseLongJumping
	PoseDying
	PoseCroaking
	PoseUsingTongue
	PoseSitting
	PoseRoaring
	PoseSniffing
	PoseEmerging
	PoseDigging
	PoseSliding
	PoseShooting
	PoseInhaling
)

// BaseEntity todo: we should wrap during load/save entities in a struct that hold the type id, exposed for nbt parser
//
//meta:encode mode=entity nbt=accessors
type BaseEntity struct {
	metadata.DirtyTracker `nbt:"-"`
	DimensionID           string                 `nbt:"-"`                                         // todo: change to a numeric id
	CustomName            proto.OptTextComponent `meta:"IndexCustomName,OptTextComponent" nbt:"-"` // todo: add text component ntb encoding
	CustomNameVisible     bool                   `meta:"IndexCustomNameVisible,Boolean" nbt:"CustomNameVisible,omitempty"`
	Motion                [3]float64
	Position              [3]float64 `nbt:"Pos"`
	ID                    ID         `nbt:"-"`
	Rotation              [2]float32
	EntityID              int32 `nbt:"-"`
	Air                   int16 `meta:"IndexAirTicks,VarInt,default=300"`
	Fire                  int16
	UUID                  NbtUUID
	Pose                  Pose `meta:"IndexPose,Pose,default=PoseStanding" nbt:"-"`
	Flags                 Flag `meta:"IndexEntityFlags,Byte,flags" nbt:"-"`
	OnGround              bool
	NoGravity             bool    `meta:"IndexNoGravity,Boolean" nbt:"NoGravity,omitempty"`
	Silent                bool    `meta:"IndexSilent,Boolean" nbt:"Silent,omitempty"`
	InSyncQueue           bool    `nbt:"-"` // used for metadata sync
	TicksFrozen           int32   `meta:"IndexTicksFrozen,VarInt" nbt:"TicksFrozen,omitempty"`
	FallDistance          float64 `nbt:"fall_distance"`
	Glowing               bool    `nbt:"Glowing,omitempty"` // this is an alias for FlagGlowing entity flag
	Invulnerable          bool
	PortalCooldown        int32
	// todo: implement Passengers, Tags and data
}

func (e *BaseEntity) GetID() int32          { return e.EntityID }
func (e *BaseEntity) GetUUID() NbtUUID      { return e.UUID }
func (e *BaseEntity) GetType() ID           { return e.ID }
func (e *BaseEntity) GetPos() [3]float64    { return e.Position }
func (e *BaseEntity) SetPos(pos [3]float64) { e.Position = pos }
func (e *BaseEntity) GetRot() [2]float32    { return e.Rotation }
func (e *BaseEntity) SetRot(rot [2]float32) { e.Rotation = rot }
func (e *BaseEntity) GetMotion() [3]float64 { return e.Motion }
func (e *BaseEntity) IsOnGround() bool      { return e.OnGround }
func (e *BaseEntity) Tick()                 {}
func (e *BaseEntity) Base() *BaseEntity     { return e }

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
