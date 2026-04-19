package parsers

import (
	"io"
	"math"
	"strconv"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type CoordKind byte

const (
	CoordAbsolute CoordKind = iota
	CoordRelative
	CoordLocal
)

type Coord struct {
	Kind  CoordKind
	Value float64
}

type ParsedVec2 struct {
	X, Z Coord
}

type ParsedVec3 struct {
	X, Y, Z Coord
}

type Vec2Type struct{}
type Vec3Type struct{}

var Vec2 = Vec2Type{}
var Vec3 = Vec3Type{}

const VecFlagIntegerOnly = 0x01

func (v Vec2Type) ID() int { return 11 }
func (v Vec3Type) ID() int { return 10 }

func (v Vec2Type) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentPos2dIncomplete),
			r.Input(), r.Cursor(),
		)
	}
	x, err := parseCoord(r, false)
	if err != nil {
		return nil, err
	}
	if err := consumeCoordSep(r, mcdata.ArgumentPos2dIncomplete); err != nil {
		return nil, err
	}
	z, err := parseCoord(r, false)
	if err != nil {
		return nil, err
	}
	return ParsedVec2{X: x, Z: z}, nil
}

func (v Vec3Type) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentPos3dIncomplete),
			r.Input(), r.Cursor(),
		)
	}
	x, err := parseCoord(r, true)
	if err != nil {
		return nil, err
	}
	if err := consumeCoordSep(r, mcdata.ArgumentPos3dIncomplete); err != nil {
		return nil, err
	}
	y, err := parseCoord(r, true)
	if err != nil {
		return nil, err
	}
	if err := consumeCoordSep(r, mcdata.ArgumentPos3dIncomplete); err != nil {
		return nil, err
	}
	z, err := parseCoord(r, true)
	if err != nil {
		return nil, err
	}

	anyLocal := x.Kind == CoordLocal || y.Kind == CoordLocal || z.Kind == CoordLocal
	allLocal := x.Kind == CoordLocal && y.Kind == CoordLocal && z.Kind == CoordLocal
	if anyLocal && !allLocal {
		return nil, commander.NewParsingError(tc.Translatable(mcdata.ArgumentPosMixed), r.Input())
	}

	return ParsedVec3{X: x, Y: y, Z: z}, nil
}

func (v Vec2Type) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

func (v Vec3Type) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

func (v ParsedVec3) Resolve(origin [3]float64, rot [2]float32) [3]float64 {
	if v.X.Kind == CoordLocal {
		return resolveLocal(v, origin, rot)
	}
	return [3]float64{
		resolveAxis(v.X, origin[0]),
		resolveAxis(v.Y, origin[1]),
		resolveAxis(v.Z, origin[2]),
	}
}

func (v ParsedVec2) Resolve(origin [3]float64) [2]float64 {
	return [2]float64{
		resolveAxis(v.X, origin[0]),
		resolveAxis(v.Z, origin[2]),
	}
}

func resolveAxis(c Coord, origin float64) float64 {
	if c.Kind == CoordAbsolute {
		return c.Value
	}
	return origin + c.Value
}

func resolveLocal(v ParsedVec3, origin [3]float64, rot [2]float32) [3]float64 {
	yaw := float64(rot[0]) * math.Pi / 180.0
	pitch := float64(rot[1]) * math.Pi / 180.0
	sy, cy := math.Sin(yaw), math.Cos(yaw)
	sp, cp := math.Sin(pitch), math.Cos(pitch)

	// forward: where the entity looks
	fx := -sy * cp
	fy := -sp
	fz := cy * cp
	ux := -sy * sp
	uy := cp
	uz := cy * sp
	lx := cy
	lz := sy

	lv, uv, fv := v.X.Value, v.Y.Value, v.Z.Value
	return [3]float64{
		origin[0] + lv*lx + uv*ux + fv*fx,
		origin[1] + uv*uy + fv*fy,
		origin[2] + lv*lz + uv*uz + fv*fz,
	}
}

func parseCoord(r *commander.CommandReader, allowLocal bool) (Coord, error) {
	if !r.CanRead() {
		return Coord{}, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentPosMissingDouble),
			r.Input(), r.Cursor(),
		)
	}

	start := r.Cursor()
	switch r.Peek() {
	case '~':
		r.Skip()
		val, err := readOptionalNumber(r, start)
		if err != nil {
			return Coord{}, err
		}
		return Coord{Kind: CoordRelative, Value: val}, nil
	case '^':
		if !allowLocal {
			return Coord{}, commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentPosMixed),
				r.Input(), start,
			)
		}
		r.Skip()
		val, err := readOptionalNumber(r, start)
		if err != nil {
			return Coord{}, err
		}
		return Coord{Kind: CoordLocal, Value: val}, nil
	default:
		if !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
			return Coord{}, commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentPosMissingDouble),
				r.Input(), start,
			)
		}
		raw := r.ReadUnquotedString()
		val, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			r.SetCursor(start)
			return Coord{}, commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ParsingDoubleInvalid, tc.Text(raw)),
				r.Input(), start,
			)
		}
		return Coord{Kind: CoordAbsolute, Value: val}, nil
	}
}

func readOptionalNumber(r *commander.CommandReader, start int) (float64, error) {
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return 0, nil
	}
	raw := r.ReadUnquotedString()
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.SetCursor(start)
		return 0, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingDoubleInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	return val, nil
}

func consumeCoordSep(r *commander.CommandReader, missingKey mcdata.TranslationKey) error {
	if !r.CanRead() || r.Peek() != ' ' {
		return commander.NewParsingErrorAt(tc.Translatable(missingKey), r.Input(), r.Cursor())
	}
	r.Skip()
	return nil
}
