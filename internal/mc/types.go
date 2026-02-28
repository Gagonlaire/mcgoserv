package mc

import (
	"context"
	"go/types"
	"io"

	"github.com/google/uuid"
)

type Field interface {
	io.ReaderFrom
	io.WriterTo
}

type (
	// Boolean Encodes:
	//  - Either false or true.
	// Size:
	//  - 1 byte
	// Notes:
	//  - True is encoded as 0x01, false as 0x00.
	Boolean bool
	// Byte Encodes:
	//  - An integer between -128 and 127.
	// Size:
	//  - 1 byte
	// Notes:
	//  - Signed 8-bit integer, two's complement.
	Byte int8
	// UnsignedByte Encodes:
	//  - An integer between 0 and 255.
	// Size:
	//  - 1 byte
	// Notes:
	//  - Unsigned 8-bit integer.
	UnsignedByte uint8
	// Short Encodes:
	//  - An integer between -32768 and 32767.
	// Size:
	//  - 2 bytes
	// Notes:
	//  - Signed 16-bit integer, two's complement.
	Short int16
	// UnsignedShort Encodes:
	//  - An integer between 0 and 65535.
	// Size:
	//  - 2 bytes
	// Notes:
	//  - Unsigned 16-bit integer
	UnsignedShort uint16
	// Int Encodes:
	//  - An integer between -2147483648 and 2147483647.
	// Size:
	//  - 4 bytes
	// Notes:
	//  - Signed 32-bit integer, two's complement.
	Int int32
	// Long Encodes:
	//  - An integer between -9223372036854775808 and 9223372036854775807.
	// Size:
	//  - 8 bytes
	// Notes:
	//  - Signed 64-bit integer, two's complement.
	Long int64
	// Float Encodes:
	//  - A single-precision 32-bit IEEE 754 floating point number.
	// Size:
	//  - 4 bytes
	Float float32
	// Double Encodes:
	//  - A double-precision 64-bit IEEE 754 floating point number.
	// Size:
	//  - 8 bytes
	Double float64
	// String (n) Encodes:
	//  - A sequence of Unicode scalar values.
	// Size:
	//  - ≥ 1 ≤ (n×3) + 3 bytes
	// Notes:
	//  - UTF-8 string prefixed with its size in bytes as a VarInt. Maximum length of n characters, which varies by context; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Type:String.
	String string
	// VarInt (n) Encodes:
	//  - An integer between -2147483648 and 2147483647.
	// Size:
	//  - ≥ 1 ≤ 5 bytes
	// Notes:
	//  - Variable-length data encoding a two's complement signed 32-bit integer; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#VarInt_and_VarLong.
	VarInt int32
	// Angle Encodes:
	//  - A rotation angle in steps of 1/256 of a full turn
	// Size:
	//  - 1 byte
	// Notes:
	//  - Whether or not this is signed does not matter, since the resulting angles are the same.
	Angle = UnsignedByte
	// UUID Encodes:
	//  - A UUID; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Type:UUID.
	// Size:
	//  - 16 bytes
	// Notes:
	//  - Encoded as an unsigned 128-bit integer (or two unsigned 64-bit integers: the most significant 64 bits and then the least significant 64 bits).
	UUID uuid.UUID
	// BitSet Encodes:
	//  - A BitSet; https://minecraft.wiki/w/Java_Edition_protocol/Data_types#BitSet.
	// Size:
	//  - Varies
	// Notes:
	//  - A length-prefixed bit set.
	BitSet struct {
		PrefixedArray[Long]
	}
	// FixedBitSet Encodes:
	//  - A FixedBitSet https://minecraft.wiki/w/Java_Edition_protocol/Packets#Fixed_BitSet.
	// Size:
	//  - ceil(n / 8)
	// Notes:
	//  - A bit set with a fixed length of n bits.
	FixedBitSet struct {
		BitCount int
		Data     []byte
	}
	// PrefixedOptional of X Encodes:
	//  - A boolean and if present, a field of type X.
	// Size:
	//  - size of Boolean + (is present ? Size of X : 0) bytes
	// Notes:
	//  - The boolean is true if the field is present.
	PrefixedOptional[X any] struct {
		Has   Boolean
		Value *X
	}
	// Array of X Encodes:
	//  - Zero or more fields of type X.
	// Size:
	//  - Length times size of X bytes
	// Notes:
	//  - The length must be known from the context.
	Array[X any] struct {
		Slice *[]X
	}
	// PrefixedArray of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Prefixed_Array.
	// Size:
	//  - size of VarInt + size of X * length bytes
	// Notes:
	//  - A length-prefixed array.
	PrefixedArray[X any] struct {
		Slice *[]X
	}
	// DataArray of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Chunk_format#Data_Array_format.
	DataArray struct {
		Slice        []uint64
		BitsPerEntry int
		Mask         uint64
		Size         int
	}
	// LpVec3 Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Data_types#LpVec3.
	// Size:
	//  - Varies
	// Notes:
	//  - Usually used for low velocities.
	LpVec3 struct {
		X, Y, Z float64
	}
	// Position Encodes:
	//  - An integer/block position: x (-33554432 to 33554431), z (-33554432 to 33554431), y (-2048 to 2047)
	// Size:
	//  - 8 bytes
	// Notes:
	//  - Encoded as a single 64-bit integer, with the x, y, and z coordinates packed into it. The x coordinate is stored in the most significant 26 bits, the z coordinate in the next 26 bits, and the y coordinate in the least significant 12 bits.
	Position struct {
		X int32
		Y int32
		Z int32
	}
)

// https://minecraft.wiki/w/Java_Edition_protocol/Packets#Player_Info_Update
type PlayerAction = UnsignedByte

type PlayerInput = UnsignedByte

type PlayerCommand int

type State int

type Pose int

type ProfileProperty struct {
	Name      string
	Value     string
	Signature string
}

type Slot struct {
	Count  int32
	ItemID int32

	Components *map[int32]any
	RemoveList *[]int32
}

// todo: create a tuple to avoid this weird struct generation

//go:generate-field-impl
type DataPackIdentifier struct {
	Namespace String
	ID        String
	Version   String
}

//go:generate-field-impl
type PlayerInformation struct {
	Locale              String
	ViewDistance        Byte
	ChatMode            VarInt
	ChatColors          Boolean // Unused by vanilla server
	DisplayedSkinParts  UnsignedByte
	MainHand            VarInt
	EnableTextFiltering Boolean
	AllowServerListings Boolean
	ParticleStatus      VarInt
}

//go:generate-field-impl
type RegistryDataEntry struct {
	ID String
	// todo: this is supposed to be an optional NTB, change later
	Data PrefixedOptional[types.Nil]
}

//go:generate-field-impl
type RegistryData struct {
	ID      String
	Entries PrefixedArray[RegistryDataEntry]
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

func (d *DataArray) ReadFrom(r io.Reader) (n int64, err error) {
	panic(context.TODO())
}

func (d *DataArray) WriteTo(w io.Writer) (n int64, err error) {
	for i := range d.Slice {
		nn, err := Long(d.Slice[i]).WriteTo(w)
		if err != nil {
			return n, err
		}
		n += nn
	}

	return n, nil
}
