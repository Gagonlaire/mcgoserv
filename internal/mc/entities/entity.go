package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/google/uuid"
)

type Entity interface {
	ID() int32
	UUID() uuid.UUID
	Type() mcdata.EntityType
	Position() [3]float64
	Tick()

	EncodeNBT() ([]byte, error)
	DecodeNBT(data []byte) error
}

// BaseEntity todo: we should use encoder functions instead of nbt struct tags
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

func (e *BaseEntity) ID() int32            { return e.EntityID }
func (e *BaseEntity) Position() [3]float64 { return e.Pos }

type LivingEntity struct {
	BaseEntity
	Health     float32
	Absorption float32
	HurtTime   int16
	DeathTime  int16
	// todo: implement attributes and effects
}

func (l *LivingEntity) IsAlive() bool { return l.Health > 0 }
