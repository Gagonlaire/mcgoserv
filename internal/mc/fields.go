package mc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"
)

func (b *Boolean) ReadFrom(r io.Reader) (n int64, err error) {
	var val byte
	if br, ok := r.(io.ByteReader); ok {
		val, err = br.ReadByte()
	} else {
		var buf [1]byte
		_, err = io.ReadFull(r, buf[:])
		val = buf[0]
	}
	if err != nil {
		return n, fmt.Errorf("error reading Boolean: %w", err)
	}
	*b = val != 0
	return 1, nil
}

func (b Boolean) WriteTo(w io.Writer) (n int64, err error) {
	val := byte(0x00)
	if b {
		val = 0x01
	}
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(val); err != nil {
			return 0, fmt.Errorf("error writing Boolean: %w", err)
		}
		return 1, nil
	}
	var buf [1]byte
	buf[0] = val
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Boolean: %w", err)
	}
	return 1, nil
}

func (b *Byte) ReadFrom(r io.Reader) (n int64, err error) {
	if br, ok := r.(io.ByteReader); ok {
		val, err := br.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("error reading Byte: %w", err)
		}
		*b = Byte(val)
		return 1, nil
	}
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Byte: %w", err)
	}
	*b = Byte(buf[0])
	return 1, nil
}

func (b Byte) WriteTo(w io.Writer) (n int64, err error) {
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(byte(b)); err != nil {
			return 0, fmt.Errorf("error writing Byte: %w", err)
		}
		return 1, nil
	}
	var buf [1]byte
	buf[0] = byte(b)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Byte: %w", err)
	}
	return 1, nil
}

func (u *UnsignedByte) ReadFrom(r io.Reader) (n int64, err error) {
	if br, ok := r.(io.ByteReader); ok {
		val, err := br.ReadByte()
		if err != nil {
			return 0, fmt.Errorf("error reading Byte: %w", err)
		}
		*u = UnsignedByte(val)
		return 1, nil
	}
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Byte: %w", err)
	}
	*u = UnsignedByte(buf[0])
	return 1, nil
}

func (u UnsignedByte) WriteTo(w io.Writer) (n int64, err error) {
	if bw, ok := w.(io.ByteWriter); ok {
		if err := bw.WriteByte(byte(u)); err != nil {
			return 0, fmt.Errorf("error writing Byte: %w", err)
		}
		return 1, nil
	}

	var buf [1]byte
	buf[0] = byte(u)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Byte: %w", err)
	}
	return 1, nil
}

func (s *Short) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Short: %w", err)
	}
	*s = Short(binary.BigEndian.Uint16(buf[:]))
	return 2, nil
}

func (s Short) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	b := binary.BigEndian.AppendUint16(buf[:0], uint16(s))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (u *UnsignedShort) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading UnsignedShort: %w", err)
	}
	*u = UnsignedShort(binary.BigEndian.Uint16(buf[:]))
	return 2, nil
}

func (u UnsignedShort) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	b := binary.BigEndian.AppendUint16(buf[:0], uint16(u))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (i *Int) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Int: %w", err)
	}
	*i = Int(binary.BigEndian.Uint32(buf[:]))
	return 4, nil
}

func (i Int) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	b := binary.BigEndian.AppendUint32(buf[:0], uint32(i))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (l *Long) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Long: %w", err)
	}
	*l = Long(binary.BigEndian.Uint64(buf[:]))
	return 8, nil
}

func (l Long) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	b := binary.BigEndian.AppendUint64(buf[:0], uint64(l))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (f *Float) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [4]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Float: %w", err)
	}
	*f = Float(math.Float32frombits(binary.BigEndian.Uint32(buf[:])))
	return 4, nil
}

func (f Float) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	b := binary.BigEndian.AppendUint32(buf[:0], math.Float32bits(float32(f)))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (d *Double) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Double: %w", err)
	}

	*d = Double(math.Float64frombits(binary.BigEndian.Uint64(buf[:])))
	return 8, nil
}

func (d Double) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	b := binary.BigEndian.AppendUint64(buf[:0], math.Float64bits(float64(d)))

	nn, err := w.Write(b)
	return int64(nn), err
}

func (s *String) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	nLen, err := length.ReadFrom(r)
	if err != nil {
		return 0, fmt.Errorf("error reading String length: %w", err)
	}
	buf := make([]byte, int(length))
	nStr, err := io.ReadFull(r, buf)
	if err != nil {
		return nLen + int64(nStr), fmt.Errorf("error reading String: %w", err)
	}
	*s = String(buf)
	return nLen + int64(nStr), nil
}

func (s String) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(s))
	n, err = length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing String length: %w", err)
	}
	nStr, err := w.Write([]byte(s))
	if err != nil {
		return n + int64(nStr), fmt.Errorf("error writing String: %w", err)
	}
	return n + int64(nStr), nil
}

func (v *VarInt) ReadFrom(r io.Reader) (n int64, err error) {
	var position uint
	*v = 0

	if br, ok := r.(io.ByteReader); ok {
		for i := 0; i < 5; i++ {
			b, err := br.ReadByte()
			if err != nil {
				return n, fmt.Errorf("error reading VarInt: %w", err)
			}
			n++
			*v |= VarInt(b&0x7F) << position
			if b&0x80 == 0 {
				return n, nil
			}
			position += 7
		}
		return n, fmt.Errorf("VarInt too long")
	}
	for i := 0; i < 5; i++ {
		var b [1]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return n, fmt.Errorf("error reading VarInt: %w", err)
		}
		n++
		*v |= VarInt(b[0]&0x7F) << position
		if b[0]&0x80 == 0 {
			return n, nil
		}
		position += 7
	}
	return n, fmt.Errorf("VarInt too long")
}

func (v VarInt) WriteTo(w io.Writer) (n int64, err error) {
	var buf [5]byte
	i := 0
	val := uint32(v)
	for {
		buf[i] = byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			buf[i] |= 0x80
		}
		i++
		if val == 0 {
			break
		}
	}
	nn, err := w.Write(buf[:i])
	return int64(nn), err
}

func (U *UUID) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, (*U)[:])
	if err != nil {
		return int64(nBytes), fmt.Errorf("error reading UUID: %w", err)
	}
	return int64(nBytes), nil
}

func (U UUID) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(U[:])
	if err != nil {
		return int64(nBytes), fmt.Errorf("error writing UUID: %w", err)
	}
	return int64(nBytes), nil
}

func (F *FixedBitSet) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, F.Data)
	return int64(nBytes), err
}

func (F FixedBitSet) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(F.Data)
	return int64(nBytes), err
}

func (p *PrefixedOptional[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	nn, err := p.Has.ReadFrom(r)
	if err != nil {
		return n, err
	}
	n += nn

	if p.Has {
		if p.Value == nil {
			p.Value = new(T)
		}
		var ptr PT = p.Value
		nn, err := ptr.ReadFrom(r)
		if err != nil {
			return n, err
		}
		n += nn
	} else {
		p.Value = nil
	}
	return n, nil
}

func (p PrefixedOptional[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	if n, err = p.Has.WriteTo(w); err != nil || !p.Has {
		return n, err
	}
	if p.Value == nil {
		return n, fmt.Errorf("invalid state: PrefixedOptional flag is true but Value is nil")
	}
	var ptr PT = p.Value
	nn, err := ptr.WriteTo(w)
	return n + nn, err
}

func (a *Array[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	for i := range a.Slice {
		var ptr PT = &a.Slice[i]
		nn, err := ptr.ReadFrom(r)
		if err != nil {
			return n, fmt.Errorf("error reading element %d of Array: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (a Array[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	for i := range a.Slice {
		var ptr PT = &a.Slice[i]
		nn, err := ptr.WriteTo(w)
		if err != nil {
			return n, fmt.Errorf("error writing element %d of Array: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (p *PrefixedArray[T, PT]) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	nn, err := length.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading PrefixedArray length: %w", err)
	}
	n += nn
	l := int(length)
	if cap(p.Slice) < l {
		p.Slice = make([]T, l)
	} else {
		p.Slice = p.Slice[:l]
	}
	for i := range p.Slice {
		var ptr PT = &p.Slice[i]
		nn, err := ptr.ReadFrom(r)
		if err != nil {
			return n, fmt.Errorf("error reading element %d of PrefixedArray: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (p PrefixedArray[T, PT]) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(p.Slice))
	nn, err := length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing PrefixedArray length: %w", err)
	}
	n += nn
	for i := range p.Slice {
		var ptr PT = &p.Slice[i]
		nn, err := ptr.WriteTo(w)
		if err != nil {
			return n, fmt.Errorf("error writing element %d of PrefixedArray: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (L *LpVec3) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return n, err
	}
	n += 1
	byte1 := uint64(buf[0])
	if byte1 == 0 {
		L.X, L.Y, L.Z = 0, 0, 0
		return n, nil
	}

	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return n, err
	}
	n += 1
	byte2 := uint64(buf[0])

	var fourBytes [4]byte
	if _, err := io.ReadFull(r, fourBytes[:]); err != nil {
		return n, err
	}
	n += 4
	bytes3To4 := binary.BigEndian.Uint32(fourBytes[:])

	packed := (uint64(bytes3To4) << 16) | (byte2 << 8) | byte1

	scaleFactor := byte1 & 3
	if (byte1 & 4) == 4 {
		rest := VarInt(0)

		nn, err := rest.ReadFrom(r)
		if err != nil {
			return n + nn, err
		}
		n += nn
		scaleFactor |= uint64(rest) << 2
	}
	scaleFactorD := float64(scaleFactor)

	L.X = unpack(packed>>3) * scaleFactorD
	L.Y = unpack(packed>>18) * scaleFactorD
	L.Z = unpack(packed>>33) * scaleFactorD

	return n, nil
}

func (L LpVec3) WriteTo(w io.Writer) (n int64, err error) {
	maxCoordinate := math.Max(math.Abs(L.X), math.Max(math.Abs(L.Y), math.Abs(L.Z)))

	if maxCoordinate < 3.051944088384301e-5 {
		if _, err := w.Write([]byte{0}); err != nil {
			return 0, err
		}
		return 1, nil
	}

	maxCoordinateI := int64(maxCoordinate)
	var scaleFactor int64
	if maxCoordinate > float64(maxCoordinateI) {
		scaleFactor = maxCoordinateI + 1
	} else {
		scaleFactor = maxCoordinateI
	}

	needContinuation := (scaleFactor & 3) != scaleFactor

	var packedScale int64
	if needContinuation {
		packedScale = (scaleFactor & 3) | 4
	} else {
		packedScale = scaleFactor
	}

	scaleFactorD := float64(scaleFactor)
	packedX := pack(L.X/scaleFactorD) << 3
	packedY := pack(L.Y/scaleFactorD) << 18
	packedZ := pack(L.Z/scaleFactorD) << 33
	packed := packedZ | packedY | packedX | packedScale

	var buf [6]byte
	buf[0] = byte(packed)
	buf[1] = byte(packed >> 8)
	valInt := uint32(packed >> 16)
	binary.BigEndian.PutUint32(buf[2:], valInt)
	if _, err := w.Write(buf[:]); err != nil {
		return 0, err
	}
	n += 6

	if needContinuation {
		buf := VarInt(scaleFactor >> 2)

		nn, err := buf.WriteTo(w)
		n += nn

		return n, err
	}

	return n, nil
}

func (s *Slot) ReadFrom(r io.Reader) (n int64, err error) {
	var count, itemID, componentToAdd, componentToRemove VarInt

	nn, err := count.ReadFrom(r)
	if err != nil {
		return nn, fmt.Errorf("error reading Slot count: %w", err)
	}
	n += nn

	s.Count = int32(count)
	if count <= 0 {
		return
	}

	nn, err = itemID.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading Slot itemID: %w", err)
	}
	n += nn
	s.ItemID = int32(itemID)

	nn, err = componentToAdd.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading Slot componentToAdd: %w", err)
	}
	n += nn
	nn, err = componentToRemove.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading Slot componentToRemove: %w", err)
	}
	n += nn

	// todo: component to add/remove should not be higher than 0 for now
	return n, nil
}

func (s Slot) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := VarInt(s.Count).WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing Slot count: %w", err)
	}
	n += nn
	if s.Count <= 0 {
		return n, nil
	}

	nn, err = VarInt(s.ItemID).WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing Slot itemID: %w", err)
	}
	n += nn

	nn, err = VarInt(0).WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing Slot componentToAdd: %w", err)
	}
	n += nn

	nn, err = VarInt(0).WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing Slot componentToRemove: %w", err)
	}
	n += nn

	return n, nil
}

func (p *ProfileProperty) ReadFrom(_ io.Reader) (int64, error) {
	panic("Not implemented")
}

func (p ProfileProperty) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := String(p.Name).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = String(p.Value).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	if p.Signature != "" {
		nn, err = Boolean(true).WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
		nn, err = String(p.Signature).WriteTo(w)
		n += nn
		return n, err
	}
	nn, err = Boolean(false).WriteTo(w)
	n += nn
	return n, err
}

func (i *Identifier) ReadFrom(r io.Reader) (n int64, err error) {
	var value String

	n, err = value.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading Identifier: %w", err)
	}

	parts := strings.Split(string(value), ":")
	count := len(parts)

	if count < 1 || count > 2 {
		return n, fmt.Errorf("invalid Identifier: too many or not enough colons in %s", value)
	}
	if count == 2 {
		if parts[0] != "minecraft" && parts[0] != "" {
			return n, fmt.Errorf("invalid Identifier: invalid namespace %s", parts[0])
		}
	}

	*i = Identifier(parts[count-1])
	for _, c := range *i {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || c == '_' || c == '-' || c == '.' || c == '/') {
			return n, fmt.Errorf("invalid Identifier: forbidden character '%c' in path %s", c, *i)
		}
	}
	return n, nil
}

func (i Identifier) WriteTo(w io.Writer) (n int64, err error) {
	// NOTE: we only support the default namespace
	return String("minecraft:" + i).WriteTo(w)
}

func (p *Position) ReadFrom(r io.Reader) (n int64, err error) {
	var val Long
	n, err = val.ReadFrom(r)
	if err != nil {
		return
	}
	p.X = int32(val >> 38)
	p.Y = int32(val << 52 >> 52)
	p.Z = int32(val << 26 >> 38)
	return
}

func (p Position) WriteTo(w io.Writer) (n int64, err error) {
	val := ((int64(p.X) & 0x3FFFFFF) << 38) | (int64(p.Y) & 0xFFF) | ((int64(p.Z) & 0x3FFFFFF) << 12)
	return Long(val).WriteTo(w)
}

func (d *DataArray) ReadFrom(_ io.Reader) (n int64, err error) {
	panic("DataArray.ReadFrom: not implemented")
}

func (d DataArray) WriteTo(w io.Writer) (n int64, err error) {
	for i := range d.Slice {
		nn, err := Long(d.Slice[i]).WriteTo(w)
		if err != nil {
			return n, err
		}
		n += nn
	}
	return n, nil
}
