package parsers

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type ResourceKey struct {
	Registry proto.Identifier
	Path     proto.Identifier
}

type ResourceKeyType struct {
	registry Registry
}

func ResourceKeyFor(registry Registry) ResourceKeyType {
	return ResourceKeyType{registry: registry}
}

func (ResourceKeyType) ID() int { return 47 } // minecraft:resource_key

func (t ResourceKeyType) Parse(r *commander.CommandReader) (any, error) {
	id, _, err := readIdentifier(r)
	if err != nil {
		return nil, err
	}
	return ResourceKey{Registry: t.registry.WireName(), Path: id}, nil
}

func (t ResourceKeyType) WriteTo(w io.Writer) (int64, error) {
	return t.registry.WireName().WriteTo(w)
}
