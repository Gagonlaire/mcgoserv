package attribute

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

func FromString(name string) (ID, bool) {
	id, ok := idByName[name]
	return id, ok
}
