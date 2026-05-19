package item

import "github.com/Gagonlaire/mcgoserv/internal/mc"

type ID uint16

func (id ID) Name() string {
	return registry[id].Name
}

func (id ID) DisplayName() string {
	return registry[id].DisplayName
}

func (id ID) StackSize() int {
	return registry[id].StackSize
}

func (id ID) BlockID() int {
	return registry[id].BlockID
}

func (id ID) EnchantCategories() []string {
	return registry[id].EnchantCategories
}

func (id ID) RepairWith() []string {
	return registry[id].RepairWith
}

func (id ID) MaxDurability() int {
	return registry[id].MaxDurability
}

func (id ID) IsDamageable() bool {
	return registry[id].MaxDurability > 0
}

func (id ID) IsBlock() bool {
	return registry[id].BlockID != -1
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

type Stack struct {
	ID     ID
	Count  int
	Damage int
}

func NewStack(id ID, count int) Stack {
	return Stack{ID: id, Count: count}
}

func (s Stack) IsEmpty() bool {
	return s.Count <= 0
}

func (s Stack) IsDamaged() bool {
	return s.Damage > 0
}

type registryImpl struct{}

// Registry exposes the item registry to the command parsers
var Registry registryImpl

func (registryImpl) WireName() mc.Identifier { return "item" }

func (registryImpl) Lookup(path mc.Identifier) (any, bool) {
	return FromString(string(path))
}
