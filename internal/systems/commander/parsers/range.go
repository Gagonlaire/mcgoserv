package parsers

import (
	"cmp"
	"io"
	"strconv"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type IntRangeType struct{}

type FloatRangeType struct{}

var IntRange = IntRangeType{}

var FloatRange = FloatRangeType{}

func (i IntRangeType) ID() int { return 39 } // minecraft:int_range

func (i IntRangeType) Parse(r *commander.CommandReader) (any, error) {
	return parseRangeArg(r, mcdata.ParsingIntInvalid, strconv.Atoi, func(min, max mc.Optional[int]) any {
		return mc.IntRange{Min: min, Max: max}
	})
}

func (i IntRangeType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (f FloatRangeType) ID() int { return 40 } // minecraft:float_range

func (f FloatRangeType) Parse(r *commander.CommandReader) (any, error) {
	return parseRangeArg(r, mcdata.ParsingDoubleInvalid, func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}, func(min, max mc.Optional[float64]) any {
		return mc.FloatRange{Min: min, Max: max}
	})
}

func (f FloatRangeType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func parseRangeArg[T cmp.Ordered](
	r *commander.CommandReader,
	invalidKey mcdata.TranslationKey,
	parse func(string) (T, error),
	build func(mc.Optional[T], mc.Optional[T]) any,
) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ArgumentRangeEmpty), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()

	rMin, rMax, err := parseRange(raw, parse)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(invalidKey, tc.Text(raw)),
			r.Input(), start,
		)
	}

	if !rMin.Present && !rMax.Present {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentRangeEmpty),
			r.Input(), start,
		)
	}

	if rMin.Present && rMax.Present && rMin.Value > rMax.Value {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentRangeSwapped),
			r.Input(), start,
		)
	}

	return build(rMin, rMax), nil
}

func parseRange[T any](raw string, parse func(string) (T, error)) (mc.Optional[T], mc.Optional[T], error) {
	var rMin, rMax mc.Optional[T]

	if idx := strings.Index(raw, ".."); idx >= 0 {
		minPart := raw[:idx]
		maxPart := raw[idx+2:]

		if len(minPart) > 0 {
			v, err := parse(minPart)
			if err != nil {
				return rMin, rMax, err
			}
			rMin = mc.Optional[T]{Value: v, Present: true}
		}
		if len(maxPart) > 0 {
			v, err := parse(maxPart)
			if err != nil {
				return rMin, rMax, err
			}
			rMax = mc.Optional[T]{Value: v, Present: true}
		}
	} else {
		v, err := parse(raw)
		if err != nil {
			return rMin, rMax, err
		}
		rMin = mc.Optional[T]{Value: v, Present: true}
		rMax = mc.Optional[T]{Value: v, Present: true}
	}

	return rMin, rMax, nil
}
