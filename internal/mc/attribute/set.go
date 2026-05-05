package attribute

import "math/rand/v2"

type Default struct {
	ID   ID
	Base float64
}

type Roll struct {
	ID       ID
	Min, Max float64
}

type Set struct {
	instances []Instance
}

func NewSet() *Set {
	return &Set{
		instances: make([]Instance, 0, 4),
	}
}

func NewSetWithDefaults(defaults []Default) *Set {
	set := &Set{
		instances: make([]Instance, len(defaults)),
	}
	for i, d := range defaults {
		set.instances[i] = Instance{
			Id:        d.ID,
			BaseValue: d.Base,
			modifiers: make([]Modifier, 0, 4),
			dirty:     true,
		}
	}
	return set
}

func (s *Set) Add(instance Instance) {
	s.instances = append(s.instances, instance)
}

func (s *Set) Get(id ID) *Instance {
	for i := range s.instances {
		if s.instances[i].Id == id {
			return &s.instances[i]
		}
	}
	return nil
}

func (s *Set) Value(id ID) float64 {
	if inst := s.Get(id); inst != nil {
		return inst.Value()
	}
	return id.Default()
}

func (s *Set) RollBases(rolls []Roll) {
	for _, r := range rolls {
		inst := s.Get(r.ID)
		if inst == nil {
			continue
		}
		inst.SetBase(r.Min + rand.Float64()*(r.Max-r.Min)) // todo: use seed based random
	}
}
