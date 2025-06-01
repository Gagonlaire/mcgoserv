package mcproto

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadVarInt read a VarInt from the provided io.Reader.
//
// Encodes:
//   - An integer between -2147483648 and 2147483647.
//
// Notes:
//   - Variable-length data encoding a two's complement signed 32-bit integer; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#VarInt_and_VarLong.
func ReadVarInt(r io.Reader) (int32, error) {
	var value int32
	var position uint

	for i := 0; i < 5; i++ {
		var b [1]byte

		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= int32(b[0]&0x7F) << position
		if b[0]&0x80 == 0 {
			return value, nil
		}
		position += 7
	}
	return 0, fmt.Errorf("VarInt trop long")
}

// WriteVarInt write a VarInt to the provided io.Writer.
func WriteVarInt(w io.Writer, value int32) error {
	for {
		temp := byte(value & 0x7F)

		value >>= 7
		if value != 0 {
			temp |= 0x80
		}
		if _, err := w.Write([]byte{temp}); err != nil {
			return err
		}
		if value == 0 {
			break
		}
	}
	return nil
}

// ReadVarLong read a VarLong from the provided io.Reader.
//
// Encodes:
//   - An integer between -9223372036854775808 and 9223372036854775807.
//
// Notes:
//   - Variable-length data encoding a two's complement signed 64-bit integer; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#VarInt_and_VarLong.
func ReadVarLong(r io.Reader) (int64, error) {
	var value int64
	var position uint
	for i := 0; i < 10; i++ {
		var b [1]byte
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, err
		}
		value |= int64(b[0]&0x7F) << position
		if b[0]&0x80 == 0 {
			return value, nil
		}
		position += 7
	}
	return 0, fmt.Errorf("VarLong trop long")
}

// WriteVarLong write a VarLong to the provided io.Writer.
func WriteVarLong(w io.Writer, value int64) error {
	for {
		temp := byte(value & 0x7F)
		value >>= 7
		if value != 0 {
			temp |= 0x80
		}
		if _, err := w.Write([]byte{temp}); err != nil {
			return err
		}
		if value == 0 {
			break
		}
	}
	return nil
}

func ReadInt(r io.Reader) (int32, error) {
	var v int32
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

func WriteInt(w io.Writer, v int32) error {
	return binary.Write(w, binary.BigEndian, v)
}

func ReadUInt16(r io.Reader) (uint16, error) {
	var v uint16
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

func WriteUInt16(w io.Writer, v uint16) error {
	return binary.Write(w, binary.BigEndian, v)
}

func ReadLong(r io.Reader) (int64, error) {
	var v int64
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}

func WriteLong(w io.Writer, v int64) error {
	return binary.Write(w, binary.BigEndian, v)
}

func ReadString(r io.Reader) (string, error) {
	length, err := ReadVarInt(r)
	if err != nil {
		return "", err
	}
	if length < 0 {
		return "", fmt.Errorf("longueur de chaîne négative")
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func WriteString(w io.Writer, s string) error {
	if err := WriteVarInt(w, int32(len(s))); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}
