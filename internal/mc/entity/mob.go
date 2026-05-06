package entity

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/attribute"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity/metadata"
	"github.com/google/uuid"
)

type Mob struct {
	LivingEntity
}

type Creature struct {
	Mob
}
type Monster struct {
	Creature
}

const (
	IndexIsBaby            metadata.Index = 16
	IndexIsBecomingDrowned metadata.Index = 18
)

//meta:encode mode=entity type=ZombieID parents=Creature nbt=accessors
type Zombie struct {
	Creature
	DrownedConversionTime int32 // todo: should default to -1
	InWaterTime           int32
	IsBaby                bool `meta:"IndexIsBaby,Boolean" nbt:"IsBaby"`
	IsBecomingDrowned     bool `meta:"IndexIsBecomingDrowned,Boolean" nbt:"-"`
	CanBreakDoors         bool
}

func NewZombie(dimension string, pos [3]float64, rot [2]float32) *Zombie {
	zombie := new(Zombie)
	zombie.DimensionID = dimension
	zombie.Position = pos
	zombie.Rotation = rot
	zombie.UUID = NbtUUID(uuid.New())
	zombie.ID = ZombieID

	zombie.InitMetaDefaults()
	zombie.Attributes = attribute.NewSetWithDefaults(attributeDefaults[ZombieID])
	zombie.Attributes.RollBases(rollableAttributes[ZombieID])
	zombie.DrownedConversionTime = -1
	zombie.Health = float32(zombie.Attributes.Value(attribute.MaxHealthID))

	return zombie
}
