package entities

import "github.com/google/uuid"

type Projectile struct {
	BaseEntity
	Owner uuid.UUID
}

type AbstractArrow struct {
	Projectile
	InGround bool
	Life     int16
	Damage   float64
	Pickup   byte
}

type Fireball struct {
	Projectile
	Power [3]float64
}

type ThrowableProjectile struct {
	Projectile
}
