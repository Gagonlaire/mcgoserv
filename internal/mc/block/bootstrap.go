package block

// RegisterAll mirrors internal/server/commands/bootstrap.go: every family adds one line here as it lands.
// Family registrations come first in alphabetical order, registerDefaultBlocks runs last
// and backfills any ID no family claimed. See README.md ("Collision resolution").
func RegisterAll() {
	registerDoors()
	registerFurnace()
	registerSlabs()
	registerDefaultBlocks()
}

func registerDefaultBlocks() {
	for i := range registry {
		if registry[i].Name == "" {
			continue
		}
		id := ID(i)
		if _, ok := Lookup(id); ok {
			continue
		}
		Register(id, NewDefaultBlock(id))
	}
}
