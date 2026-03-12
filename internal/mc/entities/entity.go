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

type BaseEntity struct {
	Dimension    any
	CustomName   string            `nbt:"CustomName,omitempty"`
	Motion       [3]float64        `nbt:"Motion"`
	Pos          [3]float64        `nbt:"Pos"`
	TypeID       mcdata.EntityType `nbt:"id"`
	Rot          [2]float32        `nbt:"Rot"`
	FallDistance float32           `nbt:"FallDistance"`
	EntityID     int32
	Fire         int16     `nbt:"Fire"`
	Air          int16     `nbt:"Air"`
	UUID         uuid.UUID `nbt:"UUID"`
	OnGround     bool      `nbt:"OnGround"`
	NoGravity    bool      `nbt:"NoGravity"`
}

func (e *BaseEntity) ID() int32            { return e.EntityID }
func (e *BaseEntity) Position() [3]float64 { return e.Pos }

type LivingEntity struct {
	BaseEntity
	Health     float32 `nbt:"Health"`
	Absorption float32 `nbt:"AbsorptionAmount"`
	HurtTime   int16   `nbt:"HurtTime"`
	DeathTime  int16   `nbt:"DeathTime"`
	// todo: implement attributes and effects
}

func (l *LivingEntity) IsAlive() bool { return l.Health > 0 }
