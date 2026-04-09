package parsers

import (
	"io"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Tnze/go-mc/nbt"
)

type ComponentType struct{}

var Component = ComponentType{}

func (c ComponentType) ID() int { return 18 } // minecraft:component

// todo: change return type to text component
func (c ComponentType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	raw, err := readSNBT(r)
	if err != nil {
		return nil, err
	}

	tagType, err := validateSNBT(raw)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentComponentInvalid),
			r.Input(), start,
		)
	}

	// A component must be either a compound a string or a list of (compound|string)
	switch tagType {
	case nbt.TagCompound, nbt.TagString, nbt.TagList:
		// todo: check list elem type or make sure parser reject them
		// todo: parse the actual component, component also support nbt path...
		return nbt.StringifiedMessage(raw), nil
	default:
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentComponentInvalid),
			r.Input(), start,
		)
	}
}

func (c ComponentType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }
