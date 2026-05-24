package world

import "fmt"

type Dimension struct {
	World  *World
	Chunks map[uint64]*Chunk
	Type   DimensionType
}

type DimensionType struct {
	CoordinateScale float64
	MinY            int
	Height          int
	AmbientLight    float32
	HasSkylight     bool
	HasCeiling      bool
	HasFixedTime    bool
}

func (d *Dimension) GetChunk(x, z int) *Chunk {
	key := (uint64(x) << 32) | (uint64(z) & 0xFFFFFFFF)
	if chunk, ok := d.Chunks[key]; ok {
		return chunk
	}

	chunk := generatePlaceholderChunk(x, z, d.Type.MinY, d.Type.Height)
	d.Chunks[key] = chunk
	return chunk
}

func (d *Dimension) GetBlock(x, y, z int) (int32, error) {
	if y < d.Type.MinY || y >= d.Type.MinY+d.Type.Height {
		return 0, fmt.Errorf("height out of bounds: %d", y)
	}

	chunkX := x >> 4
	chunkZ := z >> 4
	chunk := d.GetChunk(chunkX, chunkZ)

	return chunk.GetBlock(x&15, y, z&15)
}

func (d *Dimension) SetBlock(x, y, z int, blockState int32) error {
	if y < d.Type.MinY || y >= d.Type.MinY+d.Type.Height {
		return fmt.Errorf("height out of bounds: %d", y)
	}

	chunkX := x >> 4
	chunkZ := z >> 4
	chunk := d.GetChunk(chunkX, chunkZ)

	return chunk.SetBlock(x&15, y, z&15, blockState)
}

func (d *Dimension) tick() {}

// TODO: replace with real world generation.
// TODO: remove one section to match vanilla sea height
func generatePlaceholderChunk(x, z, minY, height int) *Chunk {
	numSections := height >> 4
	chunk := NewChunk(x, z, numSections, minY)
	const placeholderState = 9
	for i := 0; i < 9 && i < numSections; i++ {
		chunk.Fill(i, placeholderState)
	}
	return chunk
}

// DefaultDimensionsType todo: use a generator for world specific data
var (
	DefaultDimensionsType = map[string]DimensionType{
		"minecraft:overworld": {
			CoordinateScale: 1.0,
			HasSkylight:     true,
			HasCeiling:      false,
			AmbientLight:    0.0,
			HasFixedTime:    false,
			MinY:            -64,
			Height:          384,
		},
		"minecraft:the_nether": {
			CoordinateScale: 8.0,
			HasSkylight:     false,
			HasCeiling:      true,
			AmbientLight:    0.0,
			HasFixedTime:    true,
			MinY:            0,
			Height:          256,
		},
		"minecraft:the_end": {
			CoordinateScale: 1.0,
			HasSkylight:     false,
			HasCeiling:      false,
			AmbientLight:    0.0,
			HasFixedTime:    true,
			MinY:            0,
			Height:          256,
		},
	}
)
