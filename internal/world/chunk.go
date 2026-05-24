package world

import "fmt"

type Chunk struct {
	X, Z          int
	Sections      []*Section
	Light         []*LightSection
	BlockEntities []BlockEntity
	Entities      map[EntityID]struct{}
	Watchers      map[EntityID]struct{}
	minY          int
	heightmaps    Heightmaps
	hmDirty       bool
}

func NewChunk(x, z, numSections, minY int) *Chunk {
	return &Chunk{
		X:        x,
		Z:        z,
		Sections: make([]*Section, numSections),
		Light:    make([]*LightSection, numSections+2),
		Entities: make(map[EntityID]struct{}),
		Watchers: make(map[EntityID]struct{}),
		minY:     minY,
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
