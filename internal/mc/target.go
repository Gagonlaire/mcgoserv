package mc

import "github.com/google/uuid"

type TargetType int

type EntityTarget struct {
	Type     TargetType
	Name     string
	UUID     uuid.UUID
	Selector *Selector
}

type Optional[T any] struct {
	Value   T
	Present bool
}

type SelectorVariable byte

type Selector struct {
	Variable SelectorVariable
	X, Y, Z  Optional[float64]
	Distance Optional[NumberRange[float64]]
	Limit    Optional[int]
	Sort     string

	// todo: implement volume, rotations
	// todo: implement score and filters (with a type like StringFilter)
	// todo: and everything else...
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
