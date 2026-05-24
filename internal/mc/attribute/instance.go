package attribute

import (
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type Instance struct {
	modifiers   []Modifier
	BaseValue   float64
	DefaultBase float64
	cachedValue float64
	Id          ID
	dirty       bool
}

func NewInstance(id ID, baseValue float64) *Instance {
	return &Instance{
		Id:          id,
		BaseValue:   baseValue,
		DefaultBase: baseValue,
		modifiers:   make([]Modifier, 0, 4),
		dirty:       true,
	}
}

func (i *Instance) Base() float64 {
	return i.BaseValue
}

func (i *Instance) SetBase(val float64) {
	if i.BaseValue != val {
		i.BaseValue = val
		i.dirty = true
	}
}

func (i *Instance) Reset() {
	i.SetBase(i.DefaultBase)
}

func (i *Instance) indexOfModifier(id proto.Identifier) int {
	for idx := range i.modifiers {
		if i.modifiers[idx].ID == id {
			return idx
		}
	}
	return -1
}

func (i *Instance) AddModifier(m Modifier) bool {
	if i.indexOfModifier(m.ID) >= 0 {
		return false
	}
	i.modifiers = append(i.modifiers, m)
	i.dirty = true
	return true
}

func (i *Instance) RemoveModifier(id proto.Identifier) bool {
	idx := i.indexOfModifier(id)
	if idx < 0 {
		return false
	}
	// swap target with last elem, then slice off end.
	lastIdx := len(i.modifiers) - 1
	i.modifiers[idx] = i.modifiers[lastIdx]
	i.modifiers = i.modifiers[:lastIdx]
	i.dirty = true
	return true
}

// Modifier returns the modifier with the given ID, if present.
func (i *Instance) Modifier(id proto.Identifier) (Modifier, bool) {
	if idx := i.indexOfModifier(id); idx >= 0 {
		return i.modifiers[idx], true
	}
	return Modifier{}, false
}

// Value returns the computed value of the attribute, recalculating it if necessary. It applies modifiers in the following order:
// 1. AddValue modifiers are added to the base value.
// 2. AddMultipliedBase modifiers multiply the base value (after AddValue) by (1 + sum of their amounts).
// 3. AddMultipliedTotal modifiers multiply the total value (after previous steps) by (1 + each modifier's amount).
// Finally, the result is clamped between the attribute's minimum and maximum values.
func (i *Instance) Value() float64 {
	if !i.dirty {
		return i.cachedValue
	}

	value := i.BaseValue
	for _, m := range i.modifiers {
		if m.Operation == AddValue {
			value += m.Amount
		}
	}

	multipliedBaseSum := 0.0
	for _, m := range i.modifiers {
		if m.Operation == AddMultipliedBase {
			multipliedBaseSum += m.Amount
		}
	}

	value *= 1.0 + multipliedBaseSum
	for _, m := range i.modifiers {
		if m.Operation == AddMultipliedTotal {
			value *= 1.0 + m.Amount
		}
	}

	attMin := i.Id.Min()
	attMax := i.Id.Max()
	if value < attMin {
		value = attMin
	} else if value > attMax {
		value = attMax
	}
	i.cachedValue = value
	i.dirty = false

	return value
}
