package block

// RegisterAll mirrors internal/server/commands/bootstrap.go: every family adds one line here as it lands.
func RegisterAll() {
	registerDefaultBlocks()
	// TODO: registerDoors() DoorBlock family.
}

func registerDefaultBlocks() {
	for i := range registry {
		if registry[i].Name == "" {
			continue
		}
		id := ID(i)
		Register(id, NewDefaultBlock(id))
	}
}
