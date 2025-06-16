package mc

import (
	"github.com/google/uuid"
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
	DataPack      struct {
		Namespace String
		ID        String
		Version   String
	}
)

type PrefixedArray[E any] struct {
	Slice *[]E
}

func NewPrefixedArray[E any](slice *[]E) *PrefixedArray[E] {
	return &PrefixedArray[E]{
		Slice: slice,
	}
}
