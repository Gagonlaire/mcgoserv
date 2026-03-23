package mc

import "github.com/google/uuid"

type TargetType VarInt

// todo: shorter names
const (
	TargetTypePlayerName TargetType = iota
	TargetTypeUUID
	TargetTypeSelector
)

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

// todo: shorter names
const (
	SelectorVariableNearestPlayer SelectorVariable = 'p'
	SelectorVariableNearestEntity SelectorVariable = 'n'
	SelectorVariableRandomPlayer  SelectorVariable = 'r'
	SelectorVariableAllPlayers    SelectorVariable = 'a'
	SelectorVariableAllEntities   SelectorVariable = 'e'
	SelectorVariableSelf          SelectorVariable = 's'
)

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

func ValidSelectorVariable(b byte) bool {
	switch SelectorVariable(b) {
	case SelectorVariableNearestPlayer, SelectorVariableNearestEntity, SelectorVariableRandomPlayer,
		SelectorVariableAllPlayers, SelectorVariableAllEntities, SelectorVariableSelf:
		return true
	}
	return false
}
