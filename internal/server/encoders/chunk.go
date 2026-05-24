package encoders

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

type LevelChunkWithLight struct {
	X                   proto.Int
	Z                   proto.Int
	HeightMaps          proto.PrefixedArray[heightmapEntry, *heightmapEntry]
	Size                proto.VarInt
	Data                proto.Array[chunkSectionWire, *chunkSectionWire]
	BlockEntities       proto.PrefixedArray[blockEntityWire, *blockEntityWire]
	SkyLightMask        proto.BitSet
	BlockLightMask      proto.BitSet
	EmptySkyLightMask   proto.BitSet
	EmptyBlockLightMask proto.BitSet
	SkyLightArrays      proto.PrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray]
	BlockLightArrays    proto.PrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray]
}

func (l *LevelChunkWithLight) ReadFrom(_ io.Reader) (int64, error) {
	panic("LevelChunkWithLight: ReadFrom not supported")
}

func (l *LevelChunkWithLight) WriteTo(w io.Writer) (n int64, err error) {
	writers := []io.WriterTo{
		l.X, l.Z,
		l.HeightMaps,
		l.Size,
		l.Data,
		l.BlockEntities,
		l.SkyLightMask, l.BlockLightMask, l.EmptySkyLightMask, l.EmptyBlockLightMask,
		l.SkyLightArrays, l.BlockLightArrays,
	}
	for _, wt := range writers {
		nn, werr := wt.WriteTo(w)
		n += nn
		if werr != nil {
			return n, werr
		}
	}
	return n, nil
}

// TODO: check if can be move to the field generator
type chunkSectionWire struct {
	BlockCount  proto.Short
	BlockStates proto.PalettedContainer
	Biomes      proto.PalettedContainer
}

func (s *chunkSectionWire) ReadFrom(_ io.Reader) (int64, error) {
	panic("chunkSectionWire: ReadFrom not supported")
}

func (s *chunkSectionWire) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := s.BlockCount.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = s.BlockStates.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = s.Biomes.WriteTo(w)
	n += nn
	return n, err
}

type heightmapEntry struct {
	Type proto.VarInt
	Data proto.PrefixedArray[proto.Long, *proto.Long]
}

func (h *heightmapEntry) ReadFrom(_ io.Reader) (int64, error) {
	panic("heightmapEntry: ReadFrom not supported")
}

func (h *heightmapEntry) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := h.Type.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = h.Data.WriteTo(w)
	n += nn
	return n, err
}

type blockEntityWire struct {
	PackedXZ proto.UnsignedByte
	Y        proto.Short
	Type     proto.VarInt
	// TODO: NBT payload field once NBT encoding lands.
}

func (b *blockEntityWire) ReadFrom(_ io.Reader) (int64, error) {
	panic("blockEntityWire: ReadFrom not supported")
}

func (b *blockEntityWire) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := b.PackedXZ.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = b.Y.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = b.Type.WriteTo(w)
	n += nn
	return n, err
}

// NewChunkData TODO: build the wire DataArray directly from the Section impl's palette
// + indices instead of round-tripping through proto.PalettedContainer.Set.
// TODO: cache the wire payload alongside the chunk: invalidate on mutation.
// TODO: async chunk encode, move NewChunkData off the world tick.
func NewChunkData(c *world.Chunk) *LevelChunkWithLight {
	numSections := c.NumSections()
	emptyBiomes := *proto.NewPalettedContainer(0)
	sections := make([]chunkSectionWire, numSections)
	dataSize := 0

	for i := 0; i < numSections; i++ {
		sec := c.Section(i)
		var states proto.PalettedContainer
		var bc proto.Short
		if sec == nil {
			states = *proto.NewPalettedContainer(0)
		} else {
			states = sectionToWire(sec)
			bc = proto.Short(sec.BlockCount())
		}
		// TODO: per section biome storage. Encoder ships empty palette for now.
		sections[i] = chunkSectionWire{
			BlockCount:  bc,
			BlockStates: states,
			Biomes:      emptyBiomes,
		}
		dataSize += 2 + sections[i].BlockStates.Size() + sections[i].Biomes.Size()
	}

	// fow now: fully light up data, dummy.
	// TODO: implement a light engine
	skyArrays := make([]proto.PrefixedByteArray, numSections)
	for i := range skyArrays {
		data := make([]byte, 2048)
		for j := range data {
			data[j] = 0xFF
		}
		skyArrays[i] = proto.PrefixedByteArray{Data: data}
	}
	skyMaskWords := []proto.Long{0}
	for i := 0; i < numSections; i++ {
		skyMaskWords[0] |= 1 << i
	}
	skyMask := proto.BitSet{
		PrefixedArray: proto.PrefixedArray[proto.Long, *proto.Long]{Data: skyMaskWords},
	}

	return &LevelChunkWithLight{
		X:                proto.Int(c.X),
		Z:                proto.Int(c.Z),
		HeightMaps:       proto.NewPrefixedArray[heightmapEntry, *heightmapEntry](nil),
		Size:             proto.VarInt(dataSize),
		Data:             proto.Array[chunkSectionWire, *chunkSectionWire]{Data: sections},
		BlockEntities:    proto.NewPrefixedArray[blockEntityWire, *blockEntityWire](nil),
		SkyLightMask:     skyMask,
		SkyLightArrays:   proto.NewPrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray](skyArrays),
		BlockLightArrays: proto.NewPrefixedArray[proto.PrefixedByteArray, *proto.PrefixedByteArray](nil),
	}
}

func sectionToWire(sec *world.Section) proto.PalettedContainer {
	first := sec.Get(0)
	single := true
	sec.Iter(func(_ int, state int32) bool {
		if state != first {
			single = false
			return false
		}
		return true
	})
	if single {
		return *proto.NewPalettedContainer(proto.VarInt(first))
	}
	pc := proto.NewPalettedContainer(proto.VarInt(first))
	sec.Iter(func(i int, state int32) bool {
		_ = pc.Set(i, state)
		return true
	})
	return *pc
}
