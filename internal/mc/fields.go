package mc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
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
		return 1, fmt.Errorf("invalid value for Boolean: %x", buf[0])
	}
	return 1, nil
}

func (b *Boolean) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	if *b {
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

func (b *Byte) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	buf[0] = byte(*b)
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

func (u *UnsignedByte) WriteTo(w io.Writer) (n int64, err error) {
	var buf [1]byte
	buf[0] = byte(*u)
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

func (s *Short) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], uint16(*s))
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

func (u *UnsignedShort) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte
	binary.BigEndian.PutUint16(buf[:], uint16(*u))
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

func (i *Int) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], uint32(*i))
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

func (l *Long) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], uint64(*l))
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

func (f *Float) WriteTo(w io.Writer) (n int64, err error) {
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], math.Float32bits(float32(*f)))
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

func (d *Double) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], math.Float64bits(float64(*d)))
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

func (s *String) WriteTo(w io.Writer) (n int64, err error) {
	length := VarInt(len(*s))
	n, err = length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing String length: %w", err)
	}
	nStr, err := w.Write([]byte(*s))
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

func (v *VarInt) WriteTo(w io.Writer) (n int64, err error) {
	val := *v
	for {
		temp := byte(val & 0x7F)
		val >>= 7
		if val != 0 {
			temp |= 0x80
		}
		var written int
		if written, err = w.Write([]byte{temp}); err != nil {
			return n, fmt.Errorf("error writing VarInt: %w", err)
		}
		n += int64(written)
		if val == 0 {
			break
		}
	}
	return n, nil
}

func (U *UUID) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, (*U)[:])
	if err != nil {
		return int64(nBytes), fmt.Errorf("error reading UUID: %w", err)
	}
	return int64(nBytes), nil
}

func (U *UUID) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(U[:])
	if err != nil {
		return int64(nBytes), fmt.Errorf("error writing UUID: %w", err)
	}
	return int64(nBytes), nil
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
				return n, fmt.Errorf("error reading PrefixedOptional value: %w", err)
			}
			n += nn
		} else {
			typeName := reflect.TypeOf(new(X)).Elem().String()
			return n, fmt.Errorf("type %s does not implement mc.Field, cannot read optional value", typeName)
		}
	} else {
		P.Value = nil
	}
	return n, nil
}

func (P *PrefixedOptional[X]) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := P.Has.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing PrefixedOptional flag: %w", err)
	}
	n += nn
	if P.Has {
		if fieldInstance, ok := any(P.Value).(Field); ok {
			nn, err := fieldInstance.WriteTo(w)
			if err != nil {
				return n, fmt.Errorf("error writing PrefixedOptional value: %w", err)
			}
			n += nn
		} else {
			typeName := reflect.TypeOf(new(X)).Elem().String()
			return n, fmt.Errorf("type %s does not implement mc.Field, cannot write optional value", typeName)
		}
	}
	return n, nil
}

func (a *Array[X]) ReadFrom(r io.Reader) (n int64, err error) {
	//TODO implement me
	panic("implement me")
}

func (a *Array[X]) WriteTo(w io.Writer) (n int64, err error) {
	if a.Slice == nil {
		return 0, nil
	}
	currentSlice := *a.Slice
	for i := range currentSlice {
		elemAddr := &currentSlice[i]
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
	if p.Slice == nil {
		p.Slice = &[]X{}
	}
	if cap(*p.Slice) < int(length) {
		*p.Slice = make([]X, length)
	} else {
		*p.Slice = (*p.Slice)[:length]
	}
	for i := 0; i < int(length); i++ {
		elemAddr := &(*p.Slice)[i]
		fieldInstance, ok := any(elemAddr).(Field)
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

func (p *PrefixedArray[X]) WriteTo(w io.Writer) (n int64, err error) {
	if p.Slice == nil {
		p.Slice = &[]X{}
	}
	currentSlice := *p.Slice
	length := VarInt(len(currentSlice))
	nn, err := length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing PrefixedArray length: %w", err)
	}
	n += nn
	for i := range length {
		elemAddr := &currentSlice[i]
		fieldInstance, ok := any(elemAddr).(Field)
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
