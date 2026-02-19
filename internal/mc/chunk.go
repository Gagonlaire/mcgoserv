package mc

import (
	"fmt"
)

//go:generate-field-impl
type Chunk struct {
	X                   Int
	Z                   Int
	HeightMaps          PrefixedArray[HeightMap]
	Size                VarInt // todo: Size of data in bytes, need to be fixed !!
	Data                Array[ChunkSection]
	BlockEntities       PrefixedArray[BlockEntity]
	SkyLightMask        BitSet
	BlockLightMask      BitSet
	EmptySkyLightMask   BitSet
	EmptyBlockLightMask BitSet
	SkyLightArrays      PrefixedArray[PrefixedArray[Byte]]
	BlockLightArrays    PrefixedArray[PrefixedArray[Byte]]
}

//go:generate-field-impl
type HeightMap struct {
	Type VarInt
	Data PrefixedArray[Long]
}

//go:generate-field-impl
type ChunkSection struct {
	BlockCount  Short
	BlockStates PalettedContainer
	Biomes      PalettedContainer
}

//go:generate-field-impl
type BlockEntity struct {
	PackedXZ UnsignedByte
	Y        Short
	Type     VarInt
	//Data     pkt.NBTField todo: implement NBTField
}

func CreateChunk(x int, z int) *Chunk {
	// todo: implement generations
	emptyPalette := NewPalettedContainer(0)
	air := ChunkSection{
		BlockCount:  0,
		BlockStates: *emptyPalette,
		Biomes:      *emptyPalette,
	}
	dirt := ChunkSection{
		BlockCount:  4096,
		BlockStates: *NewPalettedContainer(9),
		Biomes:      *emptyPalette,
	}
	sections := make([]ChunkSection, 24)
	dataSize := 0
	i := 0
	for ; i < 9; i++ {
		sections[i] = dirt
		dataSize += 2 + sections[i].BlockStates.Size() + sections[i].Biomes.Size()
	}
	for ; i < 24; i++ {
		sections[i] = air
		dataSize += 2 + sections[i].BlockStates.Size() + sections[i].Biomes.Size()
	}

	return &Chunk{
		X:    Int(x),
		Z:    Int(z),
		Size: VarInt(dataSize),
		Data: Array[ChunkSection]{
			Slice: &sections,
		},
	}
}

func (c *Chunk) GetBlock(x, y, z, minY int) (int32, error) {
	sectionIndex := (y - minY) >> 4
	if sectionIndex < 0 || sectionIndex >= len(*c.Data.Slice) {
		return 0, fmt.Errorf("y out of bounds")
	}

	section := (*c.Data.Slice)[sectionIndex]

	relY := y & 15
	index := (relY << 8) | (z << 4) | x

	return section.BlockStates.Get(index), nil
}

func (c *Chunk) SetBlock(x, y, z, minY int, blockState int32) error {
	// todo: based on the nbt struct of chunk, minY must be stored in each sections
	sectionIndex := (y - minY) >> 4
	if sectionIndex < 0 || sectionIndex >= len(*c.Data.Slice) {
		return fmt.Errorf("y out of bounds")
	}

	section := (*c.Data.Slice)[sectionIndex]

	relY := y & 15
	index := (relY << 8) | (z << 4) | x

	// todo: handle palette resizing if necessary
	return section.BlockStates.Set(index, blockState)
}
