package sound

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
