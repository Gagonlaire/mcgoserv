package world

import (
	"fmt"
	"iter"
)

type Chunk struct {
	X, Z          int
	Sections      []*Section
	Light         []*LightSection
	blockEntities map[BlockPos]BlockEntity
	tickers       []Ticker
	Entities      map[EntityID]struct{}
	Watchers      map[EntityID]struct{}
	minY          int
	heightmaps    Heightmaps
	hmDirty       bool
}

func NewChunk(x, z, numSections, minY int) *Chunk {
	return &Chunk{
		X:             x,
		Z:             z,
		Sections:      make([]*Section, numSections),
		Light:         make([]*LightSection, numSections+2),
		blockEntities: make(map[BlockPos]BlockEntity),
		Entities:      make(map[EntityID]struct{}),
		Watchers:      make(map[EntityID]struct{}),
		minY:          minY,
	}
}

func (c *Chunk) SetBlockEntity(be BlockEntity) {
	pos := be.Pos()
	if _, exists := c.blockEntities[pos]; exists {
		c.removeTicker(pos)
	}
	c.blockEntities[pos] = be
	if t, ok := be.(Ticker); ok {
		c.tickers = append(c.tickers, t)
	}
}

// BlockEntity returns the block entity at pos, or nil if absent.
func (c *Chunk) BlockEntity(pos BlockPos) BlockEntity {
	return c.blockEntities[pos]
}

// RemoveBlockEntity removes the block entity at pos from both the map and, if it
// was a ticker, the ticker slice.
func (c *Chunk) RemoveBlockEntity(pos BlockPos) {
	if _, ok := c.blockEntities[pos]; !ok {
		return
	}
	delete(c.blockEntities, pos)
	c.removeTicker(pos)
}

// removeTicker drops the ticker at pos via swap+pop
func (c *Chunk) removeTicker(pos BlockPos) {
	for i, t := range c.tickers {
		if t.Pos() == pos {
			last := len(c.tickers) - 1
			c.tickers[i] = c.tickers[last]
			c.tickers[last] = nil
			c.tickers = c.tickers[:last]
			return
		}
	}
}

// BlockEntities iterates all block entities in this chunk without exposing the
// underlying map. Iteration order is unspecified.
func (c *Chunk) BlockEntities() iter.Seq2[BlockPos, BlockEntity] {
	return func(yield func(BlockPos, BlockEntity) bool) {
		for pos, be := range c.blockEntities {
			if !yield(pos, be) {
				return
			}
		}
	}
}

// Tick drives only the ticking block entities, never the full set.
// TODO: a ticker that places/breaks a BE in this same chunk mutates c.tickers
// mid-loop. Snapshot the slice here when the first self-mutating BE lands.
func (c *Chunk) Tick(ctx *BETickContext) {
	for _, t := range c.tickers {
		t.Tick(ctx)
	}
}

func (c *Chunk) NumSections() int { return len(c.Sections) }

func (c *Chunk) MinY() int { return c.minY }

func (c *Chunk) Section(i int) *Section {
	if i < 0 || i >= len(c.Sections) {
		return nil
	}
	return c.Sections[i]
}

func (c *Chunk) GetBlock(x, y, z int) (int32, error) {
	si := (y - c.minY) >> 4
	if si < 0 || si >= len(c.Sections) {
		return 0, fmt.Errorf("y out of bounds: %d", y)
	}
	sec := c.Sections[si]
	if sec == nil {
		return 0, nil
	}
	return sec.Get((((y - c.minY) & 15) << 8) | (z << 4) | x), nil
}

func (c *Chunk) SetBlock(x, y, z int, state int32) error {
	si := (y - c.minY) >> 4
	if si < 0 || si >= len(c.Sections) {
		return fmt.Errorf("y out of bounds: %d", y)
	}
	sec := c.Sections[si]
	if sec == nil {
		if state == 0 {
			return nil
		}
		sec = NewSection(0)
		c.Sections[si] = sec
	}
	idx := (((y - c.minY) & 15) << 8) | (z << 4) | x
	if err := sec.Set(idx, state); err != nil {
		return err
	}
	c.hmDirty = true
	return nil
}

func (c *Chunk) Fill(si int, state int32) {
	if si < 0 || si >= len(c.Sections) {
		return
	}
	if state == 0 {
		c.Sections[si] = nil
		c.hmDirty = true
		return
	}
	sec := c.Sections[si]
	if sec == nil {
		c.Sections[si] = NewSection(state)
	} else {
		sec.Fill(state)
	}
	c.hmDirty = true
}

func (c *Chunk) HeightmapsDirty() bool { return c.hmDirty }

func (c *Chunk) Heightmaps() *Heightmaps {
	if c.hmDirty {
		c.heightmaps.recompute(c)
		c.hmDirty = false
	}
	return &c.heightmaps
}
