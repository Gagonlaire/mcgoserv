package blockentity

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/container"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func Name(t Type) string {
	if int(t) < 0 || int(t) >= len(registry) {
		return ""
	}
	return registry[t].Name
}

func FromString(name string) (Type, bool) {
	t, ok := idByName[name]
	return t, ok
}

// Ticker is implemented by BEs that need per-tick work (furnace, hopper, beacon,
// conduit). It aliases world.Ticker so world.Chunk can hold []Ticker without an
// import cycle
type Ticker = world.Ticker

// Container is implemented by BEs that hold items (chest, furnace, hopper...).
type Container interface {
	world.BlockEntity
	Inventory() *container.Instance
}

type Interactable interface {
	world.BlockEntity
	OnInteract(ctx world.InteractContext) world.InteractResult
}

// Networked is implemented by BEs that store data
type Networked interface {
	world.BlockEntity
	NetworkData() any
}

// Persistent is the seam for region-file save/load.
type Persistent interface {
	world.BlockEntity
	Marshal() any
	Unmarshal(data any) error
}
