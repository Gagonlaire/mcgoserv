package mc

import (
	"fmt"
	"github.com/google/uuid"
	"io"
)

type (
	VarInt        int32
	String        string
	UnsignedShort uint16
	UUID          uuid.UUID
	Long          int64
)

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

func (U UUID) WriteTo(w io.Writer) (n int64, err error) {
	nBytes, err := w.Write(U[:])

	return int64(nBytes), err
}

func (U *UUID) ReadFrom(r io.Reader) (n int64, err error) {
	nBytes, err := io.ReadFull(r, (*U)[:])

	return int64(nBytes), err
}

func (u UnsignedShort) WriteTo(w io.Writer) (n int64, err error) {
	var buf [2]byte

	buf[0] = byte(u >> 8)
	buf[1] = byte(u & 0xFF)
	if _, err = w.Write(buf[:]); err != nil {
		return 0, err
	}

	return int64(len(buf)), nil
}

func (u *UnsignedShort) ReadFrom(r io.Reader) (n int64, err error) {
	var buf [2]byte

	if _, err = io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	*u = UnsignedShort(buf[0])<<8 | UnsignedShort(buf[1])
	n = int64(len(buf))

	return n, nil
}

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
