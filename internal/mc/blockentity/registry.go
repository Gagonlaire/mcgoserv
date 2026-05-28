package blockentity

import "github.com/Gagonlaire/mcgoserv/internal/world"

// Factory constructs a BE from a position and an NBT root
type Factory func(pos world.BlockPos, data any) (world.BlockEntity, error)

// TODO: once every BE type has a concrete implementation, switch to another query method
var factories [len(registry)]Factory

func Register(t Type, f Factory) {
	if int(t) < 0 || int(t) >= len(factories) {
		panic("blockentity.Register: type out of range")
	}
	if factories[t] != nil {
		panic("blockentity.Register: type already registered")
	}
	factories[t] = f
}

func Lookup(t Type) (Factory, bool) {
	if int(t) < 0 || int(t) >= len(factories) {
		return nil, false
	}
	f := factories[t]
	if f == nil {
		return nil, false
	}
	return f, true
}
