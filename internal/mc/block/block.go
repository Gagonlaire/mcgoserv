package block

import (
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type ID uint16

func (id ID) Name() string {
	return registry[id].Name
}

func (id ID) DisplayName() string {
	return registry[id].DisplayName
}

func (id ID) Hardness() float32 {
	return registry[id].Hardness
}

func (id ID) Resistance() float32 {
	return registry[id].Resistance
}

func (id ID) StackSize() int {
	return registry[id].StackSize
}

func (id ID) Diggable() bool {
	return registry[id].Diggable
}

func (id ID) Material() string {
	return registry[id].Material
}

func (id ID) Transparent() bool {
	return registry[id].Transparent
}

func (id ID) EmitLight() int {
	return registry[id].EmitLight
}

func (id ID) FilterLight() int {
	return registry[id].FilterLight
}

func (id ID) BoundingBox() string {
	return registry[id].BoundingBox
}

func (id ID) Drops() []int {
	return registry[id].Drops
}

func (id ID) HarvestTools() map[string]bool {
	return registry[id].HarvestTools
}

func (id ID) MinStateID() int {
	return registry[id].MinStateID
}

func (id ID) MaxStateID() int {
	return registry[id].MaxStateID
}

func (id ID) DefaultStateID() int {
	return registry[id].DefaultStateID
}

func (id ID) States() []StateProperty {
	return registry[id].States
}

func (id ID) Sounds() map[string]int {
	return registry[id].Sounds
}

func (id ID) Raw() Properties {
	return registry[id]
}

func FromString(name string) (ID, bool) {
	id, ok := idByName[name]
	return id, ok
}

func FromID(id int) (ID, bool) {
	if id < 0 || id >= len(registry) || registry[ID(id)].Name == "" {
		return 0, false
	}
	return ID(id), true
}

func FromStateID(stateID int) (ID, bool) {
	if stateID < 0 || stateID >= len(stateIDToBlock) {
		return 0, false
	}
	id := stateIDToBlock[stateID]
	if registry[id].Name == "" {
		return 0, false
	}
	return id, true
}

type registryImpl struct{}

// Registry exposes the block registry to the command parsers
var Registry registryImpl

func (registryImpl) WireName() proto.Identifier { return "block" }

func (registryImpl) Lookup(path proto.Identifier) (any, bool) {
	return FromString(string(path))
}
