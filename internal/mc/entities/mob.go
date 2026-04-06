package entities

import "github.com/google/uuid"

type Mob struct {
	LivingEntity
	NoAI       bool
	Persistent bool
	LeftHanded bool
	CanPickUp  bool
}

type PathfinderMob struct {
	Mob
}

type AgeableMob struct {
	PathfinderMob
	Age       int32
	ForcedAge int32
	AgeLocked bool
}

type Animal struct {
	AgeableMob
	InLove    int32
	LoveCause uuid.UUID
}

type TameableAnimal struct {
	Animal
	Owner uuid.UUID
	Tame  bool
}

type Monster struct {
	PathfinderMob
}

type FlyingMob struct {
	Mob
}

type WaterAnimal struct {
	PathfinderMob
}

type AmbientCreature struct {
	Mob
}

type Raider struct {
	Monster
	PatrolLeader bool
	Patrolling   bool
}

type AbstractVillager struct {
	AgeableMob
}

type AbstractHorse struct {
	Animal
	Tame bool
}

type Slime struct {
	Mob
	Size int32
}
