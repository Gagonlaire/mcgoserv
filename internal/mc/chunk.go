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
	Entities            map[EntityID]struct{} `field:"-"`
	Watchers            map[EntityID]struct{} `field:"-"`
	Data                Array[ChunkSection, *ChunkSection]
	HeightMaps          PrefixedArray[HeightMap, *HeightMap]
	BlockEntities       PrefixedArray[BlockEntity, *BlockEntity]
	SkyLightMask        BitSet
	BlockLightMask      BitSet
	EmptySkyLightMask   BitSet
	EmptyBlockLightMask BitSet
	SkyLightArrays      PrefixedArray[PrefixedArray[Byte, *Byte], *PrefixedArray[Byte, *Byte]]
	BlockLightArrays    PrefixedArray[PrefixedArray[Byte, *Byte], *PrefixedArray[Byte, *Byte]]
	Z                   Int
	Size                VarInt
	X                   Int
}

//go:generate-field-impl
type HeightMap struct {
	Data PrefixedArray[Long, *Long]
	Type VarInt
}

//go:generate-field-impl
type ChunkSection struct {
	BlockStates PalettedContainer
	Biomes      PalettedContainer
	BlockCount  Short
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
			Slice: skyMaskSlice,
		},
	}

	arrays := make([]PrefixedArray[Byte, *Byte], len(sections))
	for i := range arrays {
		data := make([]Byte, 2048)
		fullBright := UnsignedByte(0xFF)
		for j := range data {
			data[j] = Byte(fullBright)
		}
		arrays[i] = PrefixedArray[Byte, *Byte]{Slice: data}
	}

	return &Chunk{
		X: Int(x),
		Z: Int(z),

		Entities: make(map[EntityID]struct{}),
		Watchers: make(map[EntityID]struct{}),

		HeightMaps:    NewPrefixedArray[HeightMap, *HeightMap]([]HeightMap{}),
		BlockEntities: NewPrefixedArray[BlockEntity, *BlockEntity]([]BlockEntity{}),

		Size:             VarInt(dataSize),
		Data:             Array[ChunkSection, *ChunkSection]{Slice: sections},
		SkyLightMask:     skyMask,
		SkyLightArrays:   NewPrefixedArray[PrefixedArray[Byte, *Byte], *PrefixedArray[Byte, *Byte]](arrays),
		BlockLightArrays: NewPrefixedArray[PrefixedArray[Byte, *Byte], *PrefixedArray[Byte, *Byte]]([]PrefixedArray[Byte, *Byte]{}),
	}
}

func (c *Chunk) GetBlock(x, y, z, minY int) (int32, error) {
	sectionIndex := (y - minY) >> 4
	if sectionIndex < 0 || sectionIndex >= len(c.Data.Slice) {
		return 0, fmt.Errorf("y out of bounds")
	}

	section := c.Data.Slice[sectionIndex]

	relY := y & 15
	index := (relY << 8) | (z << 4) | x

	return section.BlockStates.Get(index), nil
}

func (c *Chunk) SetBlock(x, y, z, minY int, blockState int32) error {
	sectionIndex := (y - minY) >> 4

	if sectionIndex < 0 || sectionIndex >= len(c.Data.Slice) {
		return fmt.Errorf("y out of bounds")
	}

	sections := c.Data.Slice
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
	for _, section := range c.Data.Slice {
		totalSize += 2 // BlockCount size (short)
		totalSize += section.BlockStates.Size()
		totalSize += section.Biomes.Size()
	}
	c.Size = VarInt(totalSize)
}
