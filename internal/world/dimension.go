package world

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type Dimension struct {
	World  *World
	Type   DimensionType
	Chunks map[uint64]*mc.Chunk
}

type DimensionType struct {
	CoordinateScale float64
	HasSkylight     bool
	HasCeiling      bool
	AmbientLight    float32
	HasFixedTime    bool
	MinY            int
	Height          int
}

func (d *Dimension) GetChunk(x, z int) *mc.Chunk {
	key := (uint64(x) << 32) | (uint64(z) & 0xFFFFFFFF)
	if chunk, ok := d.Chunks[key]; ok {
		return chunk
	}

	chunk := mc.CreateChunk(x, z)
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

	return chunk.GetBlock(x&15, y, z&15, d.Type.MinY)
}

func (d *Dimension) SetBlock(x, y, z int, blockState int32) error {
	if y < d.Type.MinY || y >= d.Type.MinY+d.Type.Height {
		return fmt.Errorf("height out of bounds: %d", y)
	}

	chunkX := x >> 4
	chunkZ := z >> 4
	chunk := d.GetChunk(chunkX, chunkZ)

	return chunk.SetBlock(x&15, y, z&15, d.Type.MinY, blockState)
}

func (d Dimension) tick() {

}

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
