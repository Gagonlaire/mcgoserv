//go:generate go run ../../cmd/gen-field

package mc

import (
	"fmt"
)

type EntityID = int32

type ChunkPos struct {
	X int
	Z int
}

//go:generate-field-impl
type Chunk struct {
	X                   Int
	Z                   Int
	Entities            map[EntityID]struct{} `field:"-"`
	Watchers            map[EntityID]struct{} `field:"-"`
	HeightMaps          PrefixedArray[HeightMap, *HeightMap]
	Size                VarInt
	Data                Array[ChunkSection, *ChunkSection]
	BlockEntities       PrefixedArray[BlockEntity, *BlockEntity]
	SkyLightMask        BitSet
	BlockLightMask      BitSet
	EmptySkyLightMask   BitSet
	EmptyBlockLightMask BitSet
	SkyLightArrays      PrefixedArray[PrefixedByteArray, *PrefixedByteArray]
	BlockLightArrays    PrefixedArray[PrefixedByteArray, *PrefixedByteArray]
}

//go:generate-field-impl
type HeightMap struct {
	Type VarInt
	Data PrefixedArray[Long, *Long]
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

	skyMaskSlice := []Long{0}
	for i := 0; i < len(sections); i++ {
		skyMaskSlice[0] |= 1 << i
	}

	skyMask := BitSet{
		PrefixedArray: PrefixedArray[Long, *Long]{
			Data: skyMaskSlice,
		},
	}

	arrays := make([]PrefixedByteArray, len(sections))
	for i := range arrays {
		data := make([]byte, 2048)
		for j := range data {
			data[j] = 0xFF
		}
		arrays[i] = PrefixedByteArray{Data: data}
	}

	return &Chunk{
		X: Int(x),
		Z: Int(z),

		Entities: make(map[EntityID]struct{}),
		Watchers: make(map[EntityID]struct{}),

		HeightMaps:    NewPrefixedArray[HeightMap, *HeightMap]([]HeightMap{}),
		BlockEntities: NewPrefixedArray[BlockEntity, *BlockEntity]([]BlockEntity{}),

		Size:             VarInt(dataSize),
		Data:             Array[ChunkSection, *ChunkSection]{Data: sections},
		SkyLightMask:     skyMask,
		SkyLightArrays:   NewPrefixedArray[PrefixedByteArray, *PrefixedByteArray](arrays),
		BlockLightArrays: NewPrefixedArray[PrefixedByteArray, *PrefixedByteArray]([]PrefixedByteArray{}),
	}
}

func (c *Chunk) GetBlock(x, y, z, minY int) (int32, error) {
	sectionIndex := (y - minY) >> 4
	if sectionIndex < 0 || sectionIndex >= len(c.Data.Data) {
		return 0, fmt.Errorf("y out of bounds")
	}

	section := c.Data.Data[sectionIndex]

	relY := y & 15
	index := (relY << 8) | (z << 4) | x

	return section.BlockStates.Get(index), nil
}

func (c *Chunk) SetBlock(x, y, z, minY int, blockState int32) error {
	sectionIndex := (y - minY) >> 4

	if sectionIndex < 0 || sectionIndex >= len(c.Data.Data) {
		return fmt.Errorf("y out of bounds")
	}

	sections := c.Data.Data
	section := &sections[sectionIndex]
	relY := y & 15
	index := (relY << 8) | (z << 4) | x
	oldState := section.BlockStates.Get(index)

	err := section.BlockStates.Set(index, blockState)
	if err != nil {
		return err
	}

	if oldState == 0 && blockState != 0 {
		section.BlockCount++
	} else if oldState != 0 && blockState == 0 {
		section.BlockCount--
	}

	c.ComputeSize()

	return nil
}

func (c *Chunk) ComputeSize() {
	totalSize := 0
	for _, section := range c.Data.Data {
		totalSize += 2 // BlockCount size (short)
		totalSize += section.BlockStates.Size()
		totalSize += section.Biomes.Size()
	}
	c.Size = VarInt(totalSize)
}
