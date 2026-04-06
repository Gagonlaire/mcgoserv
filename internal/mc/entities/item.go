package entities

type ItemEntity struct {
	BaseEntity
	Age    int16
	Health int16
}

type ExperienceOrb struct {
	BaseEntity
	Age   int16
	Value int16
}

type FallingBlock struct {
	BaseEntity
	Time int32
}
