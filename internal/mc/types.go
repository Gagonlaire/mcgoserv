package mc

import (
	"github.com/google/uuid"
	"io"
)

type Field interface {
	io.WriterTo
	io.ReaderFrom
}

type (
	VarInt        int32
	String        string
	UnsignedShort uint16
	UUID          uuid.UUID
	Long          int64
)
