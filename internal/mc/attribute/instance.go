package attribute

type Instance struct {
	id          ID
	baseValue   float64
	modifiers   []Modifier
	cachedValue float64
	dirty       bool
}

func NewInstance(id ID) *Instance {
	return &Instance{
		id:        id,
		baseValue: id.Default(),
		modifiers: make([]Modifier, 0, 4),
		dirty:     true,
	}
}

func (i *Instance) Base() float64 {
	return i.baseValue
}

func (i *Instance) SetBase(val float64) {
	if i.baseValue != val {
		i.baseValue = val
		i.dirty = true
	}
}

func (i *Instance) AddModifier(m Modifier) {
	for idx, existing := range i.modifiers {
		if existing.ID == m.ID {
			i.modifiers[idx] = m
			i.dirty = true
			return
		}
	}

	i.modifiers = append(i.modifiers, m)
	i.dirty = true
}

func (i *Instance) RemoveModifier(id string) {
	for idx, existing := range i.modifiers {
		if existing.ID == id {
			// swap target with last elem, then slice off end.
			lastIdx := len(i.modifiers) - 1
			i.modifiers[idx] = i.modifiers[lastIdx]
			i.modifiers = i.modifiers[:lastIdx]
			i.dirty = true
			return
		}
	}
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

	value := i.baseValue
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

	attMin := i.id.Min()
	attMax := i.id.Max()
	if value < attMin {
		value = attMin
	} else if value > attMax {
		value = attMax
	}
	i.cachedValue = value
	i.dirty = false

	return value
}
