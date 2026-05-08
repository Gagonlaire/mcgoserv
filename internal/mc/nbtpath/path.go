package nbtpath

type PathStep interface {
	isStep()
}

type MemberStep struct {
	Name string
}

func (MemberStep) isStep() {}

type IndexStep struct {
	Index int
}

func (IndexStep) isStep() {}

type AllStep struct{}

func (AllStep) isStep() {}

type SelfMatch struct {
	Filter map[string]any
}

func (SelfMatch) isStep() {}

type MatchAll struct {
	Filter map[string]any
}

func (MatchAll) isStep() {}

type Path struct {
	Steps []PathStep
	Raw   string
}

type Anchor struct {
	Parent any
	Key    string
	Index  int
}
