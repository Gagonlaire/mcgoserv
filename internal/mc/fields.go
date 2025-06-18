package mc

import (
	"fmt"
	"io"
	"reflect"
)

// ReadFrom reads a Long from the provided io.Reader.
//
// Encodes:
//   - An integer between -9223372036854775808 and 9223372036854775807.
//
// Notes:
//   - Signed 64-bit integer, two's complement.
func (l *Long) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [8]byte

	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	*l = 0
	for i := 0; i < 8; i++ {
		*l = (*l << 8) | Long(buf[i])
	}

	return 8, nil
}

// WriteTo writes a Long to the provided io.Writer.
func (l Long) WriteTo(w io.Writer) (n int64, err error) {
	var buf [8]byte

	b := buf[:]
	for i := 7; i >= 0; i-- {
		b[i] = byte(l)
		l >>= 8
	}
	_, err = w.Write(b)
	if err != nil {
		return 0, err
	}

	return 8, nil
}

// ReadFrom reads a UUID from the provided io.Reader.
//
// Encodes:
//   - A UUID.
//
// Notes:
//   - Encoded as an unsigned 128-bit integer (or two unsigned 64-bit integers: the most significant 64 bits and then the least significant 64 bits).
func (U *UUID) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, (*U)[:])

	return int64(nBytes), err
}

// WriteTo writes a UUID to the provided io.Writer.
func (U UUID) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(U[:])

	return int64(nBytes), err
}

// ReadFrom reads an UnsignedShort from the provided io.Reader.
//
// Encodes:
//   - An integer between 0 and 65535.
//
// Notes:
//   - Unsigned 16-bit integer.
func (u *UnsignedShort) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte

	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	*u = UnsignedShort(buf[0])<<8 | UnsignedShort(buf[1])
	n = int64(len(buf))

	return n, nil
}

// WriteTo writes an UnsignedShort to the provided io.Writer.
func (u UnsignedShort) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte

	buf[0] = byte(u >> 8)
	buf[1] = byte(u & 0xFF)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, err
	}

	return int64(len(buf)), nil
}

// ReadFrom reads a String from the provided io.Reader.
//
// Encodes:
//   - A sequence of Unicode scalar values.
//
// Notes:
//   - UTF-8 string prefixed with its size in bytes as a VarInt. Maximum length of n characters, which varies by context; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Type:String.
func (s *String) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt
	var nLen int64

	if nLen, err = length.ReadFrom(r); err != nil {
		return 0, err
	}
	buf := make([]byte, length)
	var nStr int
	if nStr, err = io.ReadFull(r, buf); err != nil {
		return nLen + int64(nStr), err
	}
	*s = String(buf)

	return nLen + int64(nStr), nil
}

// WriteTo writes a String to the provided io.Writer.
func (s String) WriteTo(w io.Writer) (n int64, err error) {
	strBytes := []byte(s)
	length := VarInt(len(strBytes))

	if n, err = length.WriteTo(w); err != nil {
		return
	}
	var nStr int
	if nStr, err = w.Write(strBytes); err != nil {
		return n + int64(nStr), err
	}

	return n + int64(nStr), nil
}

// ReadFrom read a VarInt from the provided io.Reader.
//
// Encodes:
//   - An integer between -2147483648 and 2147483647.
//
// Notes:
//   - Variable-length data encoding a two's complement signed 32-bit integer; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#VarInt_and_VarLong.
func (v *VarInt) ReadFrom(r io.Reader) (n int64, err error) {
	var position uint

	for i := 0; i < 5; i++ {
		var b [1]byte
		var read int

		if read, err = io.ReadFull(r, b[:]); err != nil {
			return n, err
		}
		n += int64(read)
		*v |= VarInt(b[0]&0x7F) << position
		if b[0]&0x80 == 0 {
			return
		}
		position += 7
	}

	return n, fmt.Errorf("VarInt trop long")
}

// WriteTo writes a VarInt to the provided io.Writer.
func (v VarInt) WriteTo(w io.Writer) (n int64, err error) {
	for {
		temp := byte(v & 0x7F)

		v >>= 7
		if v != 0 {
			temp |= 0x80
		}
		var written int
		if written, err = w.Write([]byte{temp}); err != nil {
			return
		}
		n += int64(written)
		if v == 0 {
			break
		}
	}

	return
}

func (v VarInt) Len() int {
	val := uint32((v << 1) ^ (v >> 31))
	n := 1

	for val >= 0x80 {
		val >>= 7
		n++
	}

	return n
}

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
	written, err := w.Write(buf[:])
	if err != nil {
		return int64(written), fmt.Errorf("error writing Boolean: %w", err)
	}

	return int64(written), nil
}

func (p *PArray[E]) ReadFrom(r io.Reader) (n int64, err error) {
	var length VarInt

	nn, err := length.ReadFrom(r)
	if err != nil {
		return n, fmt.Errorf("error reading prefixed array length: %w", err)
	}
	n += nn

	if cap(*p.Slice) < int(length) {
		*p.Slice = make([]E, length)
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
			return n, fmt.Errorf("error reading element %d of prefixed array: %w", i, err)
		}
		n += nn
	}

	return n, nil
}

func (p *PArray[E]) WriteTo(w io.Writer) (n int64, err error) {
	currentSlice := *p.Slice
	length := VarInt(len(currentSlice))

	nn, err := length.WriteTo(w)
	if err != nil {
		return n, fmt.Errorf("error writing prefixed array length: %w", err)
	}
	n += nn

	for i := range currentSlice {
		elemAddr := &currentSlice[i]

		fieldInstance, ok := any(elemAddr).(Field)
		if !ok {
			typeName := reflect.TypeOf(elemAddr).String()
			return n, fmt.Errorf("element of type %s does not implement mc.Field required for writing", typeName)
		}

		nn, err := fieldInstance.WriteTo(w)
		if err != nil {
			return n, fmt.Errorf("error writing element %d of prefixed array: %w", i, err)
		}
		n += nn
	}

	return n, nil
}

func (P *POptional[E]) ReadFrom(r io.Reader) (n int64, err error) {
	if _, err := P.Has.ReadFrom(r); err != nil {
		return n, err
	}
	n += 1

	if P.Has {
		if fieldInstance, ok := any(&P.Value).(Field); ok {
			nn, err := fieldInstance.ReadFrom(r)
			if err != nil {
				return n, err
			}
			n += nn
		} else {
			return n, nil
		}
	}

	return n, nil
}

func (P *POptional[E]) WriteTo(w io.Writer) (n int64, err error) {
	if nn, err := P.Has.WriteTo(w); err != nil {
		return n, err
	} else {
		n += nn
	}

	if P.Has {
		if fieldInstance, ok := any(&P.Value).(Field); ok {
			if nn, err := fieldInstance.WriteTo(w); err != nil {
				return n, err
			} else {
				n += nn
			}
		}
	}

	return n, nil
}
