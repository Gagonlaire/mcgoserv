package entities

import "github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"

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

//meta:encode mode=entity type=EntityZombie parents=Creature nbt=accessors
type Zombie struct {
	Creature
	DrownedConversionTime int32
	InWaterTime           int32
	IsBaby                bool `meta:"IndexIsBaby,Boolean" nbt:"IsBaby"`
	IsBecomingDrowned     bool `meta:"IndexIsBecomingDrowned,Boolean" nbt:"-"`
	CanBreakDoors         bool
}
