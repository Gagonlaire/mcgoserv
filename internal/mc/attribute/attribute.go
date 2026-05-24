package attribute

import (
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type Operation int

const (
	// AddValue adds all of the modifiers' amounts to the base attribute.
	AddValue Operation = iota
	// AddMultipliedBase multiplies the base attribute by (1 + sum of modifiers' amounts).
	AddMultipliedBase
	// AddMultipliedTotal multiplies the base attribute by (1 + modifiers' amounts) for every modifier.
	AddMultipliedTotal
)

type Modifier struct {
	ID        proto.Identifier
	Amount    float64
	Operation Operation
}

type ID uint16

func (id ID) String() string {
	return registry[id].Name
}

func (id ID) Default() float64 {
	return registry[id].Default
}

func (id ID) Min() float64 {
	return registry[id].Min
}

func (id ID) Max() float64 {
	return registry[id].Max
}

func (id ID) Raw() Properties {
	return registry[id]
}

func FromString(name string) (ID, bool) {
	id, ok := idByName[name]
	return id, ok
}

type registryImpl struct{}

// Registry exposes the attribute registry to the command parsers
var Registry registryImpl

func (registryImpl) WireName() proto.Identifier { return "attribute" }

func (registryImpl) Lookup(path proto.Identifier) (any, bool) {
	return FromString(string(path))
}
