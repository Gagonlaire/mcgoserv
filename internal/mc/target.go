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
	Sort       Optional[string]
	Distance   Optional[FloatRange]
	X, Y, Z    Optional[float64]
	Dx, Dy, Dz Optional[float64]
	XRotation  Optional[FloatRange]
	YRotation  Optional[FloatRange]
	Limit      Optional[int]
	Level      Optional[IntRange]
	Gamemode   Optional[string]
	Variable   SelectorVariable
}

type IntRange struct {
	Min Optional[int]
	Max Optional[int]
}

type FloatRange struct {
	Min Optional[float64]
	Max Optional[float64]
}

type SelectorSpan struct {
	Start, End int
	Selector   *Selector
}

type ParsedMessage struct {
	Raw       string
	Selectors []SelectorSpan
}

func ValidSelectorVariable(b byte) bool {
	switch SelectorVariable(b) {
	case SelectorVariableNearestPlayer, SelectorVariableNearestEntity, SelectorVariableRandomPlayer,
		SelectorVariableAllPlayers, SelectorVariableAllEntities, SelectorVariableSelf:
		return true
	}
	return false
}
