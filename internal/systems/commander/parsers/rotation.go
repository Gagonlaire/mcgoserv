package parsers

import (
	"io"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type ParsedRotation struct {
	Yaw, Pitch Coord
}

type RotationType struct{}

var Rotation = RotationType{}

func (RotationType) ID() int { return 29 } // minecraft:rotation

func (RotationType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingError(r, tc.Translatable(mcdata.ArgumentRotationIncomplete))
	}
	yaw, err := parseCoord(r, false)
	if err != nil {
		return nil, err
	}
	if err := consumeCoordSep(r, mcdata.ArgumentRotationIncomplete); err != nil {
		return nil, err
	}
	pitch, err := parseCoord(r, false)
	if err != nil {
		return nil, err
	}
	return ParsedRotation{Yaw: yaw, Pitch: pitch}, nil
}

func (RotationType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

// Resolve applies any `~`-relative axes against the origin rotation [yaw, pitch].
func (rot ParsedRotation) Resolve(origin [2]float32) [2]float32 {
	return [2]float32{
		float32(resolveAxis(rot.Yaw, float64(origin[0]))),
		float32(resolveAxis(rot.Pitch, float64(origin[1]))),
	}
}
