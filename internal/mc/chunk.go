//go:generate go run ../../cmd/gen-field

package mc

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type EntityID = int32

type ChunkPos struct {
	X int
	Z int
}

//field:encode mode=both
type Chunk struct {
	X                   proto.Int
	Z                   proto.Int
	Entities            map[EntityID]struct{} `field:"-"`
	Watchers            map[EntityID]struct{} `field:"-"`
	HeightMaps          proto.PrefixedArray[HeightMap, *HeightMap]
	Size                proto.VarInt
	Data                proto.Array[ChunkSection, *ChunkSection]
	BlockEntities       proto.PrefixedArray[BlockEntity, *BlockEntity]
	SkyLightMask        proto.BitSet
	BlockLightMask      proto.BitSet
	EmptySkyLightMask   proto.BitSet
	EmptyBlockLightMask proto.BitSet
	SkyLightArrays      proto.PrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray]
	BlockLightArrays    proto.PrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray]
}

//field:encode mode=both
type HeightMap struct {
	Type proto.VarInt
	Data proto.PrefixedArray[proto.Long, *proto.Long]
}

//field:encode mode=both
type ChunkSection struct {
	BlockCount  proto.Short
	BlockStates proto.PalettedContainer
	Biomes      proto.PalettedContainer
}

//field:encode mode=both
type BlockEntity struct {
	PackedXZ proto.UnsignedByte
	Y        proto.Short
	Type     proto.VarInt
	//Data     pkt.NBTField todo: implement NBTField
}

func CreateChunk(x int, z int) *Chunk {
	// todo: implement generations
	emptyPalette := proto.NewPalettedContainer(0)
	air := ChunkSection{
		BlockCount:  0,
		BlockStates: *emptyPalette,
		Biomes:      *emptyPalette,
	}
	dirt := ChunkSection{
		BlockCount:  4096,
		BlockStates: *proto.NewPalettedContainer(9),
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

	skyMaskSlice := []proto.Long{0}
	for i := 0; i < len(sections); i++ {
		skyMaskSlice[0] |= 1 << i
	}

	skyMask := proto.BitSet{
		PrefixedArray: proto.PrefixedArray[proto.Long, *proto.Long]{
			Data: skyMaskSlice,
		},
	}

	arrays := make([]proto.PrefixedByteArray, len(sections))
	for i := range arrays {
		data := make([]byte, 2048)
		for j := range data {
			data[j] = 0xFF
		}
		arrays[i] = proto.PrefixedByteArray{Data: data}
	}

	return &Chunk{
		X: proto.Int(x),
		Z: proto.Int(z),

		Entities: make(map[EntityID]struct{}),
		Watchers: make(map[EntityID]struct{}),

		HeightMaps:    proto.NewPrefixedArray[HeightMap, *HeightMap]([]HeightMap{}),
		BlockEntities: proto.NewPrefixedArray[BlockEntity, *BlockEntity]([]BlockEntity{}),

		Size:             proto.VarInt(dataSize),
		Data:             proto.Array[ChunkSection, *ChunkSection]{Data: sections},
		SkyLightMask:     skyMask,
		SkyLightArrays:   proto.NewPrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray](arrays),
		BlockLightArrays: proto.NewPrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray]([]proto.PrefixedByteArray{}),
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
	c.Size = proto.VarInt(totalSize)
}
