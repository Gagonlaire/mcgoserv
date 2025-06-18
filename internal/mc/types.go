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
	VarInt        int32
	String        string
	UnsignedShort uint16
	UUID          uuid.UUID
	Long          int64
	Boolean       bool
	// look if we can modify any with a list of mc types, to be less error-prone
	PArray[E any] struct {
		Slice *[]E
	}
	POptional[E any] struct {
		Has   Boolean
		Value E
	}
)

// Following types use auto-generated code to implement Field interface.

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
	Data POptional[*types.Nil]
}

//go:generate-field-impl
type RegistryData struct {
	ID      String
	Entries PArray[RegistryDataEntry]
}

func NewPArray[E any](slice *[]E) *PArray[E] {
	return &PArray[E]{
		Slice: slice,
	}
}

// todo: like PArray, Value should be a pointer to avoid copying large structs, memory issues and potential errors
func NewPOptional[E any](value E) *POptional[E] {
	return &POptional[E]{
		Has:   false,
		Value: value,
	}
}
