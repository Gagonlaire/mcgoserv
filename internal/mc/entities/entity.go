package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/google/uuid"
)

type Entity interface {
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

type BaseEntity struct {
	Dimension    any
	CustomName   string
	Motion       [3]float64
	Pos          [3]float64
	TypeID       mcdata.EntityType
	Rot          [2]float32
	FallDistance float32
	EntityID     int32
	Fire         int16
	Air          int16
	UUID         uuid.UUID
	OnGround     bool
	NoGravity    bool
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
