package mc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
)

func (b *Boolean) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Boolean: %w", err)
	}
	switch buf[0] {
	case 0x00:
		*b = false
	case 0x01:
		*b = true
	default:
		return 1, fmt.Errorf("invalid Value for Boolean: %x", buf[0])
	}
	return 1, nil
}

func (b Boolean) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	if b {
		buf[0] = 0x01
	} else {
		buf[0] = 0x00
	}
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Boolean: %w", err)
	}
	return 1, nil
}

func (b *Byte) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading Byte: %w", err)
	}
	*b = Byte(int8(buf[0]))
	return 1, nil
}

func (b Byte) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	buf[0] = byte(b)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Byte: %w", err)
	}
	return 1, nil
}

func (u *UnsignedByte) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [1]byte
	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("error reading UnsignedByte: %w", err)
	}
	*u = UnsignedByte(buf[0])
	return 1, nil
}

func (u UnsignedByte) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	buf[0] = byte(u)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing UnsignedByte: %w", err)
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
	binary.BigEndian.PutUint16(buf[:], uint16(s))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Short: %w", err)
	}
	return 2, nil
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
	binary.BigEndian.PutUint16(buf[:], uint16(u))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing UnsignedShort: %w", err)
	}
	return 2, nil
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
	binary.BigEndian.PutUint32(buf[:], uint32(i))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Int: %w", err)
	}
	return 4, nil
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
	binary.BigEndian.PutUint64(buf[:], uint64(l))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Long: %w", err)
	}
	return 8, nil
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
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(float32(f)))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Float: %w", err)
	}

	return 4, nil
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
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(float64(d)))
	if _, err = w.Write(buf[:]); err != nil {
		return 0, fmt.Errorf("error writing Double: %w", err)
	}
	return 8, nil
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
	for i := 0; i < 5; i++ {
		var b [1]byte
		var read int

		if read, err = io.ReadFull(r, b[:]); err != nil {
			return n, fmt.Errorf("error reading VarInt: %w", err)
		}
		n += int64(read)
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

func (P *PrefixedOptional[X]) ReadFrom(r io.Reader) (n int64, err error) {
	nn, err := P.Has.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading PrefixedOptional flag: %w", err)
	}
	n += nn
	if P.Has {
		if fieldInstance, ok := any(P.Value).(Field); ok {
			nn, err := fieldInstance.ReadFrom(r)
			if err != nil {
				return n, fmt.Errorf("error reading PrefixedOptional Value: %w", err)
			}
			n += nn
		} else {
			typeName := reflect.TypeOf(new(X)).Elem().String()
			return n, fmt.Errorf("type %s does not implement mc.Field, cannot read optional Value", typeName)
		}
	} else {
		P.Value = nil
	}
	return n, nil
}

func (P PrefixedOptional[X]) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := P.Has.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing PrefixedOptional flag: %w", err)
	}
	n += nn
	if P.Has {
		if fieldInstance, ok := any(P.Value).(Field); ok {
			nn, err := fieldInstance.WriteTo(w)
			if err != nil {
				return n, fmt.Errorf("error writing PrefixedOptional Value: %w", err)
			}
			n += nn
		} else {
			typeName := reflect.TypeOf(new(X)).Elem().String()
			return n, fmt.Errorf("type %s does not implement mc.Field, cannot write optional Value", typeName)
		}
	}
	return n, nil
}

// ReadFrom todo: this should not need reflection type check and accept the field interface directly
func (a *Array[X]) ReadFrom(r io.Reader) (n int64, err error) {
	for i := range a.Slice {
		elemAddr := &a.Slice[i]
		fieldInstance, ok := any(elemAddr).(Field)
		if !ok {
			typeName := reflect.TypeOf(elemAddr).String()
			return n, fmt.Errorf("element of type %s does not implement mc.Field required for reading", typeName)
		}
		nn, err := fieldInstance.ReadFrom(r)
		if err != nil {
			return n, fmt.Errorf("error reading element %d of Array: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (a Array[X]) WriteTo(w io.Writer) (n int64, err error) {
	for i := range a.Slice {
		elemAddr := &a.Slice[i]
		fieldInstance, ok := any(elemAddr).(Field)
		if !ok {
			typeName := reflect.TypeOf(elemAddr).String()
			return n, fmt.Errorf("element of type %s does not implement mc.Field required for writing", typeName)
		}
		nn, err := fieldInstance.WriteTo(w)
		if err != nil {
			return n, fmt.Errorf("error writing element %d of Array: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (p *PrefixedArray[X]) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	nn, err := length.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading PrefixedArray length: %w", err)
	}
	n += nn
	if cap(p.Slice) < int(length) {
		p.Slice = make([]X, length)
	} else {
		p.Slice = p.Slice[:length]
	}
	for i := 0; i < int(length); i++ {
		elemAddr := &p.Slice[i]
		fieldInstance, ok := any(elemAddr).(io.ReaderFrom)
		if !ok {
			typeName := reflect.TypeOf(elemAddr).String()
			return n, fmt.Errorf("element of type %s does not implement mc.Field required for reading", typeName)
		}
		nn, err := fieldInstance.ReadFrom(r)
		if err != nil {
			return n, fmt.Errorf("error reading element %d of PrefixedArray: %w", i, err)
		}
		n += nn
	}
	return n, nil
}

func (p PrefixedArray[X]) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(p.Slice))
	nn, err := length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing PrefixedArray length: %w", err)
	}
	n += nn
	for i := range length {
		elemAddr := &p.Slice[i]
		fieldInstance, ok := any(elemAddr).(io.WriterTo)
		if !ok {
			typeName := reflect.TypeOf(elemAddr).String()
			return n, fmt.Errorf("element of type %s does not implement mc.Field required for writing", typeName)
		}
		nn, err := fieldInstance.WriteTo(w)
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
