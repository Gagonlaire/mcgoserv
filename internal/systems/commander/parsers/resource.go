package parsers

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type ResourceType struct {
	registry Registry
}

func Resource(registry Registry) ResourceType {
	return ResourceType{registry: registry}
}

func (ResourceType) ID() int { return 46 } // minecraft:resource

func (t ResourceType) Parse(r *commander.CommandReader) (any, error) {
	id, start, err := readIdentifier(r)
	if err != nil {
		return nil, err
	}
	value, ok := t.registry.Lookup(id)
	if !ok {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentResourceNotFound, tc.Text(string(id)), tc.Text(string(t.registry.WireName()))),
			r.Input(), start,
		)
	}
	return value, nil
}

func (t ResourceType) WriteTo(w io.Writer) (int64, error) {
	return mc.Identifier(t.registry.WireName()).WriteTo(w)
}
