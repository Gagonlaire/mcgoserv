package mc

import (
	"io"
)

const (
	MinIndirectBits = 4
	MaxIndirectBits = 8
	DirectBits      = 15
	SectionSize     = 4096
)

type Container interface {
	Field

	// Get returns the Value at the given index (0-4095).
	Get(index int) int32

	// Set tries to set the Value at the given index (0-4095), in some cases, it may change de palette type.
	Set(index int, value int32) (c Container, err error)

	// Size returns the size of the container in bytes.
	Size() int

	// BitsPerEntry returns the number of bits used per Value.
	BitsPerEntry() UnsignedByte
}

// PalettedContainer todo: implement upgrade logic for containers
type PalettedContainer struct {
	c Container
}

//field:encode mode=both
type SingleValueContainer struct {
	Value VarInt
}

//field:encode mode=both
type IndirectContainer struct {
	DataArray   *DataArray
	Palette     PrefixedArray[VarInt, *VarInt]
	MaxCapacity int `field:"-"`
}

//field:encode mode=both
type DirectContainer struct {
	DataArray *DataArray
}

func NewPalettedContainer(singlePaletteValue VarInt) *PalettedContainer {
	return &PalettedContainer{
		c: newSingleValueContainer(singlePaletteValue),
	}
}

func (p *PalettedContainer) ReadFrom(_ io.Reader) (n int64, err error) {
	panic("Not implemented")
}

func (p *PalettedContainer) WriteTo(w io.Writer) (n int64, err error) {
	bpe := p.c.BitsPerEntry()

	nn, err := bpe.WriteTo(w)
	if err != nil {
		return nn, err
	}
	n += nn
	nn, err = p.c.WriteTo(w)
	if err != nil {
		return n, err
	}
	n += nn
	return n, nil
}

func (p *PalettedContainer) Get(index int) int32 {
	return p.c.Get(index)
}

func (p *PalettedContainer) Set(index int, value int32) error {
	newImpl, err := p.c.Set(index, value)
	if err != nil {
		return err
	}
	if newImpl != nil {
		p.c = newImpl
	}
	return nil
}

func (p *PalettedContainer) Size() int {
	// Container "Size" method only returns the size of the data in the container,
	// so we add 1 byte for the bits per entry.
	return 1 + p.c.Size()
}

func newSingleValueContainer(value VarInt) *SingleValueContainer {
	return &SingleValueContainer{Value: value}
}

func (s *SingleValueContainer) Get(_ int) int32 {
	return int32(s.Value)
}

func (s *SingleValueContainer) Set(index int, value int32) (Container, error) {
	if int32(s.Value) == value {
		return nil, nil
	}

	indirect := newIndirectContainer(MinIndirectBits)
	paletteData := []VarInt{s.Value}
	indirect.Palette = NewPrefixedArray[VarInt, *VarInt](paletteData)
	_, err := indirect.Set(index, value)
	if err != nil {
		return nil, err
	}

	return indirect, nil
}

func (s *SingleValueContainer) Size() int {
	return s.Value.Len()
}

func (s *SingleValueContainer) BitsPerEntry() UnsignedByte { return 0 }

func newIndirectContainer(bpe int) *IndirectContainer {
	return &IndirectContainer{
		MaxCapacity: 1 << bpe,
		Palette:     NewPrefixedArray[VarInt, *VarInt](make([]VarInt, 0)),
		DataArray:   NewDataArray(bpe, SectionSize),
	}
}

func (i *IndirectContainer) Get(index int) int32 {
	paletteIndex := i.DataArray.Get(index)
	if paletteIndex >= len(i.Palette.Data) {
		return 0
	}
	return int32(i.Palette.Data[paletteIndex])
}

func (i *IndirectContainer) Set(index int, value int32) (Container, error) {
	pIndex := -1
	for idx, v := range i.Palette.Data {
		if int32(v) == value {
			pIndex = idx
			break
		}
	}

	if pIndex != -1 {
		i.DataArray.Set(index, pIndex)
		return nil, nil
	}

	if len(i.Palette.Data) >= i.MaxCapacity {
		newBPE := i.DataArray.BitsPerEntry + 1
		if newBPE > MaxIndirectBits {
			return i.upgradeToDirect(index, value)
		}
		i.MaxCapacity = 1 << newBPE
		i.resize(newBPE)
	}

	i.Palette.Data = append(i.Palette.Data, VarInt(value))
	i.DataArray.Set(index, len(i.Palette.Data)-1)

	return nil, nil
}

func (i *IndirectContainer) resize(newBPE int) {
	newStorage := NewDataArray(newBPE, SectionSize)

	for x := 0; x < SectionSize; x++ {
		val := i.DataArray.Get(x)
		newStorage.Set(x, val)
	}
	i.DataArray = newStorage
}

func (i *IndirectContainer) upgradeToDirect(index int, value int32) (Container, error) {
	direct := NewDirectContainer()

	for x := 0; x < SectionSize; x++ {
		globalID := i.Get(x)
		_, _ = direct.Set(x, globalID)
	}
	return direct.Set(index, value)
}

func (i *IndirectContainer) Size() int {
	pSize := VarInt(len(i.Palette.Data)).Len()
	for _, v := range i.Palette.Data {
		pSize += v.Len()
	}
	dataLen := len(i.DataArray.Data)

	return pSize + dataLen*8
}

func (i *IndirectContainer) BitsPerEntry() UnsignedByte {
	return UnsignedByte(i.DataArray.BitsPerEntry)
}

func NewDirectContainer() *DirectContainer {
	return &DirectContainer{
		DataArray: NewDataArray(DirectBits, SectionSize),
	}
}

func (d *DirectContainer) Get(index int) int32 {
	return int32(d.DataArray.Get(index))
}

func (d *DirectContainer) Set(index int, value int32) (Container, error) {
	d.DataArray.Set(index, int(value))
	return nil, nil
}

func (d *DirectContainer) Size() int {
	return len(d.DataArray.Data) * 8
}

func (d *DirectContainer) BitsPerEntry() UnsignedByte {
	return UnsignedByte(DirectBits)
}
