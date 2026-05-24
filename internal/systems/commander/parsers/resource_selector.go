package parsers

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

// ResourceSelector todo: replace Pattern with a parsed selector when wildcard implemented
type ResourceSelector struct {
	Registry proto.Identifier
	Pattern  proto.Identifier
}

type ResourceSelectorType struct {
	registry Registry
}

func ResourceSelectorFor(registry Registry) ResourceSelectorType {
	return ResourceSelectorType{registry: registry}
}

func (ResourceSelectorType) ID() int { return 48 } // minecraft:resource_selector

func (t ResourceSelectorType) Parse(r *commander.CommandReader) (any, error) {
	id, _, err := readIdentifier(r)
	if err != nil {
		return nil, err
	}
	return ResourceSelector{Registry: t.registry.WireName(), Pattern: id}, nil
}

func (t ResourceSelectorType) WriteTo(w io.Writer) (int64, error) {
	return t.registry.WireName().WriteTo(w)
}
