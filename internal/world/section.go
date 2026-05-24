package world

import "fmt"

const (
	// SectionSize is the number of cells in a 16×16×16 block section.
	SectionSize = 4096
	// indirectMaxPaletteSize is the in-memory cap for Indirect mode (uint8 indices).
	indirectMaxPaletteSize = 256
)

type Section struct {
	blockCount uint16
	impl       sectionImpl
}

type sectionImpl interface {
	get(idx int) int32
	set(idx int, state int32) (next sectionImpl, old int32)
	fill(state int32) sectionImpl
	fillRange(yStart, yEnd int, state int32) sectionImpl
	iter(yield func(idx int, state int32) bool)
}

func NewSection(state int32) *Section {
	s := &Section{impl: newSingleImpl(state)}
	if state != 0 {
		s.blockCount = SectionSize
	}
	return s
}

func (s *Section) BlockCount() uint16 { return s.blockCount }

func (s *Section) Get(idx int) int32 {
	if idx < 0 || idx >= SectionSize {
		return 0
	}
	return s.impl.get(idx)
}

func (s *Section) Set(idx int, state int32) error {
	if idx < 0 || idx >= SectionSize {
		return fmt.Errorf("section index out of range: %d", idx)
	}
	next, old := s.impl.set(idx, state)
	if next != nil {
		s.impl = next
	}
	if old == 0 && state != 0 {
		s.blockCount++
	} else if old != 0 && state == 0 {
		s.blockCount--
	}
	if s.blockCount == 0 {
		s.impl = newSingleImpl(0)
	}
	return nil
}

func (s *Section) Fill(state int32) {
	s.impl = newSingleImpl(state)
	if state == 0 {
		s.blockCount = 0
	} else {
		s.blockCount = SectionSize
	}
}

func (s *Section) FillRange(yStart, yEnd int, state int32) {
	if yStart < 0 {
		yStart = 0
	}
	if yEnd > 16 {
		yEnd = 16
	}
	if yStart >= yEnd {
		return
	}
	if yStart == 0 && yEnd == 16 {
		s.Fill(state)
		return
	}
	const slabCells = 16 * 16

	oldNonAir := 0
	for y := yStart; y < yEnd; y++ {
		base := y << 8
		for i := 0; i < slabCells; i++ {
			if s.impl.get(base+i) != 0 {
				oldNonAir++
			}
		}
	}
	if next := s.impl.fillRange(yStart, yEnd, state); next != nil {
		s.impl = next
	}
	newNonAir := 0
	if state != 0 {
		newNonAir = (yEnd - yStart) * slabCells
	}
	s.blockCount = s.blockCount - uint16(oldNonAir) + uint16(newNonAir)
	if s.blockCount == 0 {
		s.impl = newSingleImpl(0)
	}
}

func (s *Section) Iter(yield func(idx int, state int32) bool) {
	s.impl.iter(yield)
}

type singleImpl struct {
	value int32
}

func newSingleImpl(value int32) *singleImpl { return &singleImpl{value: value} }

func (s *singleImpl) get(_ int) int32 { return s.value }

func (s *singleImpl) set(idx int, state int32) (sectionImpl, int32) {
	if state == s.value {
		return nil, s.value
	}
	indirect := &indirectImpl{
		palette: []int32{s.value, state},
		indices: make([]uint8, SectionSize),
	}
	indirect.indices[idx] = 1
	return indirect, s.value
}

func (s *singleImpl) fill(state int32) sectionImpl { return newSingleImpl(state) }

func (s *singleImpl) fillRange(yStart, yEnd int, state int32) sectionImpl {
	if state == s.value {
		return nil
	}
	indirect := &indirectImpl{
		palette: []int32{s.value, state},
		indices: make([]uint8, SectionSize),
	}
	for y := yStart; y < yEnd; y++ {
		base := y << 8
		for i := 0; i < 256; i++ {
			indirect.indices[base+i] = 1
		}
	}
	return indirect
}

func (s *singleImpl) iter(yield func(idx int, state int32) bool) {
	for i := 0; i < SectionSize; i++ {
		if !yield(i, s.value) {
			return
		}
	}
}

type indirectImpl struct {
	palette []int32
	indices []uint8
}

func (i *indirectImpl) get(idx int) int32 {
	return i.palette[i.indices[idx]]
}

func (i *indirectImpl) set(idx int, state int32) (sectionImpl, int32) {
	old := i.palette[i.indices[idx]]
	for pi, ps := range i.palette {
		if ps == state {
			i.indices[idx] = uint8(pi)
			return nil, old
		}
	}
	if len(i.palette) >= indirectMaxPaletteSize {
		direct := newDirectImpl()
		for k, ix := range i.indices {
			direct.states[k] = uint16(i.palette[ix])
		}
		direct.states[idx] = uint16(state)
		return direct, old
	}
	i.palette = append(i.palette, state)
	i.indices[idx] = uint8(len(i.palette) - 1)
	return nil, old
}

func (i *indirectImpl) fill(state int32) sectionImpl { return newSingleImpl(state) }

func (i *indirectImpl) fillRange(yStart, yEnd int, state int32) sectionImpl {
	pi := -1
	for k, ps := range i.palette {
		if ps == state {
			pi = k
			break
		}
	}
	if pi < 0 {
		if len(i.palette) >= indirectMaxPaletteSize {
			direct := newDirectImpl()
			for k, ix := range i.indices {
				direct.states[k] = uint16(i.palette[ix])
			}
			for y := yStart; y < yEnd; y++ {
				base := y << 8
				for k := 0; k < 256; k++ {
					direct.states[base+k] = uint16(state)
				}
			}
			return direct
		}
		i.palette = append(i.palette, state)
		pi = len(i.palette) - 1
	}
	pu := uint8(pi)
	for y := yStart; y < yEnd; y++ {
		base := y << 8
		for k := 0; k < 256; k++ {
			i.indices[base+k] = pu
		}
	}
	return nil
}

func (i *indirectImpl) iter(yield func(idx int, state int32) bool) {
	for k, ix := range i.indices {
		if !yield(k, i.palette[ix]) {
			return
		}
	}
}

type directImpl struct {
	states [SectionSize]uint16
}

func newDirectImpl() *directImpl { return &directImpl{} }

func (d *directImpl) get(idx int) int32 { return int32(d.states[idx]) }

func (d *directImpl) set(idx int, state int32) (sectionImpl, int32) {
	old := int32(d.states[idx])
	d.states[idx] = uint16(state)
	return nil, old
}

func (d *directImpl) fill(state int32) sectionImpl { return newSingleImpl(state) }

func (d *directImpl) fillRange(yStart, yEnd int, state int32) sectionImpl {
	for y := yStart; y < yEnd; y++ {
		base := y << 8
		for k := 0; k < 256; k++ {
			d.states[base+k] = uint16(state)
		}
	}
	return nil
}

func (d *directImpl) iter(yield func(idx int, state int32) bool) {
	for k, st := range d.states {
		if !yield(k, int32(st)) {
			return
		}
	}
}

// TODO: pool []uint8 index arrays for Indirect impls via sync.Pool.
// TODO: full palette compaction with per-entry usage counts.
// TODO: cache packed wire DataArray alongside Section and invalidate on Set.
