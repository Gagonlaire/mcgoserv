package sound

import "github.com/Gagonlaire/mcgoserv/internal/mc"

type ID uint16

func (id ID) Name() string {
	return registry[id].Name
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

type registryImpl struct{}

// Registry exposes the sound-event registry to the command parsers
var Registry registryImpl

func (registryImpl) WireName() mc.Identifier { return "sound_event" }

func (registryImpl) Lookup(path mc.Identifier) (any, bool) {
	return FromString(string(path))
}
