package entity

import "fmt"

type ID uint16

func (id ID) Name() string        { return registry[id].Name }
func (id ID) DisplayName() string { return registry[id].DisplayName }
func (id ID) Width() float64      { return registry[id].Width }
func (id ID) Height() float64     { return registry[id].Height }
func (id ID) Type() string        { return registry[id].Type }
func (id ID) Category() string    { return registry[id].Category }
func (id ID) Raw() Properties     { return registry[id] }

func (id ID) MarshalText() ([]byte, error) {
	if int(id) >= len(registry) || registry[id].Name == "" {
		return nil, fmt.Errorf("unknown entity ID %d", id)
	}
	return []byte(registry[id].Name), nil
}

func (id *ID) UnmarshalText(text []byte) error {
	v, ok := idByName[string(text)]
	if !ok {
		return fmt.Errorf("unknown entity type %s", text)
	}
	*id = v
	return nil
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
