package entities

type LivingEntity struct {
	BaseEntity
	Health     float32
	Absorption float32
	HurtTime   int16
	DeathTime  int16
}
