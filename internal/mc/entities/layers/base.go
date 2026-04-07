package layers

type Tracker interface {
	MarkDirty(index byte)
	IsDirty(index byte) bool
}

type BaseLayer struct {
	tracker Tracker
}

func (b *BaseLayer) Init(t Tracker) {
	b.tracker = t
}

func (b *BaseLayer) markDirty(index byte) {
	if b.tracker != nil {
		b.tracker.MarkDirty(index)
	}
}

func (b *BaseLayer) isDirty(index byte) bool {
	if b.tracker != nil {
		return b.tracker.IsDirty(index)
	}
	return false
}
