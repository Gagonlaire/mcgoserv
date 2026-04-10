package mc

import (
	"fmt"
	"io"
	"iter"

	"github.com/Gagonlaire/mcgoserv/internal/errutil"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
)

type (
	// BitSet Encodes:
	//  - A BitSet; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#BitSet.
	// Size:
	//  - Varies
	// Notes:
	//  - A length-prefixed bit set.
	BitSet struct {
		PrefixedArray[Long, *Long]
	}
	// FixedBitSet Encodes:
	//  - A FixedBitSet https://minecraft.wiki/w/Java_Edition_protocol/Packets#Fixed_BitSet.
	// Size:
	//  - ceil(n / 8)
	// Notes:
	//  - A bit set with a fixed length of n bits.
	FixedBitSet struct {
		Data     []byte
		BitCount int
	}
	// PrefixedOptional of X Encodes:
	//  - A boolean and if present, a field of type X.
	// Size:
	//  - size of Boolean + (is present ? Size of X : 0) bytes
	// Notes:
	//  - The boolean is true if the field is present.
	PrefixedOptional[T any, PT FieldPtr[T]] struct {
		Value T
		Has   Boolean
	}
	// Array of X Encodes:
	//  - Zero or more fields of type X.
	// Size:
	//  - Length times size of X bytes
	// Notes:
	//  - The length must be known from the context.
	Array[T any, PT FieldPtr[T]] struct {
		Data []T
	}
	// PrefixedArray of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Prefixed_Array.
	// Size:
	//  - size of VarInt + size of X * length bytes
	// Notes:
	//  - A length-prefixed array.
	PrefixedArray[T any, PT FieldPtr[T]] struct {
		Data      []T
		MaxLength int32
	}
	// ByteArray Encodes:
	//  - Depends on context.
	// Size:
	//  - Varies
	// Notes:
	//  - This is just a sequence of zero or more bytes, its meaning should be explained somewhere else, e.g. in the packet description. The length must also be known from the context.
	ByteArray struct {
		Data []byte
	}
	// PrefixedByteArray not a protocol type, just useful in certain contexts
	PrefixedByteArray struct {
		Data      []byte
		MaxLength int32
	}
	// IDOrX of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Packets#ID_or_X.
	// Size:
	//  - size of VarInt + (size of X or 0)
	// Notes:
	//  - Either a registry ID or an inline data definition of type X.
	IDOrX[T any, PT FieldPtr[T]] struct {
		Data    T
		ID      int32
		HasData bool
	}
	// DataArray of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Chunk_format#Data_Array_format.
	DataArray struct {
		Data         []uint64
		BitsPerEntry int
		Mask         uint64
		Size         int
	}
)

// Component is an interface, it cannot satisfy  FieldPtr, so we need to define a separate types for it
type (
	OptTextComponent struct {
		Value tc.Component
		Has   bool
	}
	IdOrTextComponent struct {
		Inline   tc.Component
		ID       int32
		IsInline bool
	}
)

func NewArray[T any, PT FieldPtr[T]](size uint32) Array[T, PT] {
	return Array[T, PT]{
		Data: make([]T, size),
	}
}

// NewByteArray creates a ByteArray with a pre-allocated Data slice of the given size.
func NewByteArray(size uint32) ByteArray {
	return ByteArray{Data: make([]byte, size)}
}

// NewPrefixedByteArray wraps an existing byte slice in a PrefixedByteArray.
func NewPrefixedByteArray(data []byte) PrefixedByteArray {
	return PrefixedByteArray{Data: data}
}

// NewPrefixedArray wraps an existing slice in a PrefixedArray.
func NewPrefixedArray[T any, PT FieldPtr[T]](data []T) PrefixedArray[T, PT] {
	return PrefixedArray[T, PT]{Data: data}
}

func NewPrefixedOptional[T any, PT FieldPtr[T]](val T) PrefixedOptional[T, PT] {
	return PrefixedOptional[T, PT]{
		Has:   true,
		Value: val,
	}
}

func NewFixedBitSet(n int) FixedBitSet {
	return FixedBitSet{
		BitCount: n,
		Data:     make([]byte, (n+7)/8),
	}
}

func NewDataArray(bitsPerEntry int, size int) *DataArray {
	valuesPerLong := 64 / bitsPerEntry
	longCount := (size + valuesPerLong - 1) / valuesPerLong

	return &DataArray{
		Data:         make([]uint64, longCount),
		BitsPerEntry: bitsPerEntry,
		Mask:         (1 << bitsPerEntry) - 1,
		Size:         size,
	}
}

// MapToPrefixedArray converts a slice of one type to a PrefixedArray of another type using a conversion function.
// ex: convert []byte to PrefixedArray[Byte]
func MapToPrefixedArray[E any, PE FieldPtr[E], S any](data []S, convert func(S) E) PrefixedArray[E, PE] {
	if data == nil {
		return PrefixedArray[E, PE]{}
	}
	newSlice := make([]E, len(data))
	for i, v := range data {
		newSlice[i] = convert(v)
	}
	return PrefixedArray[E, PE]{Data: newSlice}
}

// CollectToPrefixedArray creates a new PrefixedArray from an iterator with a conversion function and filtering.
// ex: iter over connections map and return the player (when the connection has one)
func CollectToPrefixedArray[E any, PE FieldPtr[E], S any](seq iter.Seq[S], convert func(S) (E, bool)) PrefixedArray[E, PE] {
	var newSlice []E
	for v := range seq {
		if mapped, keep := convert(v); keep {
			newSlice = append(newSlice, mapped)
		}
	}
	return PrefixedArray[E, PE]{Data: newSlice}
}

// MapToSlice converts a PrefixedArray to a regular slice using a conversion function.
// ex: convert PrefixedArray[Byte] to []byte
func MapToSlice[E any, PE FieldPtr[E], T any](p PrefixedArray[E, PE], convert func(E) T) []T {
	if p.Data == nil {
		return nil
	}
	dst := make([]T, len(p.Data))
	for i, v := range p.Data {
		dst[i] = convert(v)
	}
	return dst
}

func (p *PrefixedOptional[T, PT]) Set(value T) {
	p.Value = value
	p.Has = true
}

func (p *PrefixedOptional[T, PT]) Clear() {
	var zero T
	p.Value = zero
	p.Has = false
}

func (b *BitSet) Set(i int, value bool) {
	data := b.Data
	idx := i / 64
	off := uint(i % 64)
	if idx >= len(data) {
		return
	}
	if value {
		data[idx] |= 1 << off
	} else {
		data[idx] &^= 1 << off
	}
}

func (b *BitSet) Get(i int) bool {
	data := b.Data
	idx := i / 64
	off := uint(i % 64)
	if idx >= len(data) {
		return false
	}
	return (data[idx] & (1 << off)) != 0
}

func (f *FixedBitSet) Set(i int, value bool) {
	if value {
		f.Data[i/8] |= 1 << (i % 8)
	} else {
		f.Data[i/8] &^= 1 << (i % 8)
	}
}

func (f *FixedBitSet) Get(i int) (bool, error) {
	return (f.Data[i/8] & (1 << (i % 8))) != 0, nil
}

func (d *DataArray) Set(index int, value int) {
	if index < 0 || index >= d.Size {
		return
	}

	valuesPerLong := 64 / d.BitsPerEntry
	cellIndex := index / valuesPerLong
	bitIndex := (index % valuesPerLong) * d.BitsPerEntry

	d.Data[cellIndex] = (d.Data[cellIndex] &^ (d.Mask << bitIndex)) | (uint64(value) & d.Mask << bitIndex)
}

func (d *DataArray) Get(index int) int {
	if index < 0 || index >= d.Size {
		return 0
	}

	valuesPerLong := 64 / d.BitsPerEntry
	cellIndex := index / valuesPerLong
	bitIndex := (index % valuesPerLong) * d.BitsPerEntry

	return int((d.Data[cellIndex] >> bitIndex) & d.Mask)
}

func (f *FixedBitSet) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, f.Data)
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error reading FixedBitSet")
	}
	return int64(nBytes), nil
}

func (f FixedBitSet) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(f.Data)
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error writing FixedBitSet")
	}
	return int64(nBytes), nil
}

func (p *PrefixedOptional[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	nn, err := p.Has.ReadFrom(r)
	n += nn
	if err != nil {
		return n, err
	}
	if p.Has {
		var ptr PT = &p.Value
		nn, err := ptr.ReadFrom(r)
		n += nn
		if err != nil {
			return n, err
		}
	} else {
		var zero T
		p.Value = zero
	}
	return n, nil
}

func (p PrefixedOptional[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	if n, err = p.Has.WriteTo(w); err != nil || !p.Has {
		return n, err
	}
	var ptr PT = &p.Value
	nn, err := ptr.WriteTo(w)
	return n + nn, err
}

func (a *Array[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	for i := range a.Data {
		var ptr PT = &a.Data[i]
		nn, err := ptr.ReadFrom(r)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (a Array[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	for i := range a.Data {
		var ptr PT = &a.Data[i]
		nn, err := ptr.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p *PrefixedArray[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	nn, err := length.ReadFrom(r)
	n += nn
	if err != nil {
		return n, err
	}
	if p.MaxLength > 0 && int32(length) > p.MaxLength {
		return n, fmt.Errorf("PrefixedArray length %d exceeds maximum length %d", length, p.MaxLength)
	}
	l := int(length)
	if cap(p.Data) < l {
		p.Data = make([]T, l)
	} else {
		p.Data = p.Data[:l]
	}
	for i := range p.Data {
		var ptr PT = &p.Data[i]
		nn, err := ptr.ReadFrom(r)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p PrefixedArray[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(p.Data))
	nn, err := length.WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	for i := range p.Data {
		var ptr PT = &p.Data[i]
		nn, err := ptr.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (a *ByteArray) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, a.Data)
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error reading ByteArray")
	}
	return int64(nBytes), nil
}

func (a ByteArray) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(a.Data)
	if err != nil {
		return int64(nBytes), errutil.WrapIOErr(err, "error writing ByteArray")
	}
	return int64(nBytes), nil
}

func (p *PrefixedByteArray) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	nn, err := length.ReadFrom(r)
	n += nn
	if err != nil {
		return nn, err
	}
	if p.MaxLength > 0 && int32(length) > p.MaxLength {
		return n, fmt.Errorf("PrefixedByteArray length %d exceeds maximum length %d", length, p.MaxLength)
	}
	l := int(length)
	if cap(p.Data) < l {
		p.Data = make([]byte, l)
	} else {
		p.Data = p.Data[:l]
	}
	nBytes, err := io.ReadFull(r, p.Data)
	if err != nil {
		return n + int64(nBytes), errutil.WrapIOErr(err, "error reading PrefixedByteArray data")
	}
	return n + int64(nBytes), nil
}

func (p PrefixedByteArray) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(p.Data))
	nn, err := length.WriteTo(w)
	n += nn
	if err != nil {
		return nn, err
	}
	nBytes, err := w.Write(p.Data)
	if err != nil {
		return n + int64(nBytes), errutil.WrapIOErr(err, "error writing PrefixedByteArray data")
	}
	return n + int64(nBytes), nil
}

func (i *IDOrX[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	var id VarInt
	nn, err := id.ReadFrom(r)
	n += nn
	if err != nil {
		return nn, err
	}
	if id == 0 {
		i.HasData = true
		var ptr PT = &i.Data
		nn, err = ptr.ReadFrom(r)
		n += nn
		if err != nil {
			return n, err
		}
		i.ID = 0
	} else {
		i.HasData = false
		var zero T
		i.Data = zero
		i.ID = int32(id) - 1
	}
	return n, nil
}

func (i IDOrX[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	if i.HasData {
		nn, err := VarInt(0).WriteTo(w)
		n += nn
		if err != nil {
			return nn, err
		}
		var ptr PT = &i.Data
		nn, err = ptr.WriteTo(w)
		return n + nn, err
	}
	return VarInt(i.ID + 1).WriteTo(w)
}

func (d *DataArray) ReadFrom(_ io.Reader) (n int64, err error) {
	panic("DataArray.ReadFrom: not implemented")
}

func (d DataArray) WriteTo(w io.Writer) (n int64, err error) {
	for i := range d.Data {
		nn, err := Long(d.Data[i]).WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func (o OptTextComponent) WriteTo(w io.Writer) (int64, error) {
	n, err := Boolean(o.Has).WriteTo(w)
	if err != nil || !o.Has {
		return n, err
	}
	nn, err := o.Value.WriteTo(w)
	return n + nn, err
}

func (r IdOrTextComponent) WriteTo(w io.Writer) (int64, error) {
	if r.IsInline {
		n, err := VarInt(0).WriteTo(w)
		if err != nil {
			return n, err
		}
		nn, err := r.Inline.WriteTo(w)
		return n + nn, err
	}
	return VarInt(r.ID + 1).WriteTo(w)
}
