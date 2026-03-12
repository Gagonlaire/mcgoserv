package mc

import "github.com/google/uuid"

type TargetType int

type EntityTarget struct {
	Selector *Selector
	Name     string
	Type     TargetType
	UUID     uuid.UUID
}

type Optional[T any] struct {
	Value   T
	Present bool
}

type SelectorVariable byte

type Selector struct {
	Sort     string
	Distance Optional[NumberRange[float64]]
	X        Optional[float64]
	Y        Optional[float64]
	Z        Optional[float64]
	Limit    Optional[int]
	Variable SelectorVariable
}

type NumberRange[T int | float64] struct {
	Min Optional[T]
	Max Optional[T]
}

const (
	SelectorNearestPlayer SelectorVariable = 'p'
	SelectorNearestEntity SelectorVariable = 'n'
	SelectorRandomPlayer  SelectorVariable = 'r'
	SelectorAllPlayers    SelectorVariable = 'a'
	SelectorAllEntities   SelectorVariable = 'e'
	SelectorSelf          SelectorVariable = 's'
)

const (
	TargetTypePlayerName TargetType = iota
	TargetTypeUUID
	TargetTypeSelector
)

func ValidSelectorVariable(b byte) bool {
	switch SelectorVariable(b) {
	case SelectorNearestPlayer, SelectorNearestEntity, SelectorRandomPlayer,
		SelectorAllPlayers, SelectorAllEntities, SelectorSelf:
		return true
	}
	return false
}
