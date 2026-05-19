package parsers

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type ResourceLocationType struct{}

var ResourceLocation = ResourceLocationType{}

func (ResourceLocationType) ID() int { return 36 } // minecraft:resource_location

func (ResourceLocationType) Parse(r *commander.CommandReader) (any, error) {
	id, _, err := readIdentifier(r)
	if err != nil {
		return nil, err
	}
	return id, nil
}

func (ResourceLocationType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }
