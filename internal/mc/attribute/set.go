package attribute

type Set struct {
	instances []*Instance // todo: using values might be better
}

func NewSet(initialAttributes ...ID) *Set {
	s := &Set{
		instances: make([]*Instance, 0, len(initialAttributes)),
	}
	for _, id := range initialAttributes {
		s.instances = append(s.instances, NewInstance(id))
	}

	return s
}

func (s *Set) Get(id ID) *Instance {
	for _, instance := range s.instances {
		if instance.id == id {
			return instance
		}
	}
	newInstance := NewInstance(id)
	s.instances = append(s.instances, newInstance)

	return newInstance
}

func (s *Set) Value(id ID) float64 {
	return s.Get(id).Value()
}
