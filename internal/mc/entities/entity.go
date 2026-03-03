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
	// Data tags
	Motion       [3]float64        `nbt:"Motion"`
	NoGravity    bool              `nbt:"NoGravity"`
	OnGround     bool              `nbt:"OnGround"`
	Pos          [3]float64        `nbt:"Pos"`
	Rot          [2]float32        `nbt:"Rot"`
	UUID         uuid.UUID         `nbt:"UUID"`
	TypeID       mcdata.EntityType `nbt:"id"`
	FallDistance float32           `nbt:"FallDistance"`
	Fire         int16             `nbt:"Fire"`
	Air          int16             `nbt:"Air"`
	CustomName   string            `nbt:"CustomName,omitempty"`

	// State
	EntityID  int32
	Dimension any // todo: maybe change to a array index or a more direct and less ambiguous access
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
