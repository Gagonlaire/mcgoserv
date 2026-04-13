package entities

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/Tnze/go-mc/nbt"
	"github.com/google/uuid"
)

type NbtUUID uuid.UUID

func (u NbtUUID) TagType() byte {
	return nbt.TagIntArray
}

func (u NbtUUID) MarshalNBT(w io.Writer) error {
	// 1. TagIntArray payload starts with the array length (number of ints)
	if err := binary.Write(w, binary.BigEndian, int32(4)); err != nil {
		return err
	}

	// 2. Convert the 16 bytes into 4 int32s
	v1 := int32(binary.BigEndian.Uint32(u[0:4]))
	v2 := int32(binary.BigEndian.Uint32(u[4:8]))
	v3 := int32(binary.BigEndian.Uint32(u[8:12]))
	v4 := int32(binary.BigEndian.Uint32(u[12:16]))

	// 3. Write the integers to the stream
	return binary.Write(w, binary.BigEndian, []int32{v1, v2, v3, v4})
}

func (u NbtUUID) UnmarshalNBT(tagType byte, r nbt.DecoderReader) error {
	if tagType != nbt.TagIntArray {
		return fmt.Errorf("expected TagIntArray (11), got %d", tagType)
	}

	// 1. Read the array length
	var length int32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return err
	}
	if length != 4 {
		return fmt.Errorf("expected UUID array length 4, got %d", length)
	}

	// 2. Read the 4 int32s
	var ints [4]int32
	if err := binary.Read(r, binary.BigEndian, &ints); err != nil {
		return err
	}

	// 3. Populate the byte array
	binary.BigEndian.PutUint32(u[0:4], uint32(ints[0]))
	binary.BigEndian.PutUint32(u[4:8], uint32(ints[1]))
	binary.BigEndian.PutUint32(u[8:12], uint32(ints[2]))
	binary.BigEndian.PutUint32(u[12:16], uint32(ints[3]))
	return nil
}
