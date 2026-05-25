package entity

type Hand uint8

// TODO: check if can be merged with the definition in living entity
const (
	MainHand Hand = iota
	OffHand
)
