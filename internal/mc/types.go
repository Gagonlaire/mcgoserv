package mc

import (
	"github.com/google/uuid"
	"go/types"
	"io"
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
	UnsignedByte byte
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
	// PrefixedArray of X Encodes:
	//  - https://minecraft.wiki/w/Java_Edition_protocol/Data_types#Prefixed_Array.
	// Size:
	//  - size of VarInt + size of X * length bytes
	// Notes:
	//  - A length-prefixed array.
	PrefixedArray[X any] struct {
		Slice *[]X
	}
)

//go:generate-field-impl
type DataPackIdentifier struct {
	Namespace String
	ID        String
	Version   String
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
