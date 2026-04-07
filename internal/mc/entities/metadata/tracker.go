package metadata

type DirtyTracker struct {
	bits uint64
}

func (d *DirtyTracker) Mark(index byte) {
	d.bits |= 1 << index
}

func (d *DirtyTracker) IsDirty(index byte) bool {
	return d.bits&(1<<index) != 0
}

func (d *DirtyTracker) HasChanges() bool {
	return d.bits != 0
}

func (d *DirtyTracker) Clear() {
	d.bits = 0
}

func (d *DirtyTracker) MarkAll() {
	d.bits = ^uint64(0)
}
