package mc

import "io"

type Container interface {
	Field

	// Get returns the value at the given index (0-4095).
	Get(index int) int32

	// Set tries to set the value at the given index (0-4095), in some cases, it may change de palette type.
	Set(index int, value int32) (c Container, err error)

	// Size returns the size of the container in bytes.
	Size() int

	// BitsPerEntry returns the number of bits used per value.
	BitsPerEntry() UnsignedByte
}

// PalettedContainer todo: implement upgrade logic for containers
type PalettedContainer struct {
	c Container
}

func (p *PalettedContainer) ReadFrom(_ io.Reader) (n int64, err error) {
	return 0, nil
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

//go:generate-field-impl
type SingleValueContainer struct {
	value VarInt
}

func NewPalettedContainer(singlePaletteValue VarInt) *PalettedContainer {
	return &PalettedContainer{
		c: newSingleValueContainer(singlePaletteValue),
	}
}

func (p *PalettedContainer) Get(index int) int32 {
	return p.c.Get(index)
}

func (p *PalettedContainer) Set(index int, value int32) error {
	newImpl, err := p.c.Set(index, value)
	if err != nil {
		return err
	}
	p.c = newImpl
	return nil
}

func (p *PalettedContainer) Size() int {
	// Container "Size" method only returns the size of the data in the container,
	// so we add 1 byte for the bits per entry.
	return 1 + p.c.Size()
}

func newSingleValueContainer(value VarInt) *SingleValueContainer {
	return &SingleValueContainer{value: value}
}

func (s *SingleValueContainer) Get(_ int) int32 {
	return int32(s.value)
}

func (s *SingleValueContainer) Set(_ int, value int32) (Container, error) {
	s.value = VarInt(value)

	return nil, nil
}

func (s *SingleValueContainer) Size() int {
	return s.value.Len()
}

func (s *SingleValueContainer) BitsPerEntry() UnsignedByte { return 0 }
