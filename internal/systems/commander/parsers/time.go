package parsers

import (
	"io"
	"math"
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type TimeType struct {
	min int32
}

var Time = TimeType{}

func (t TimeType) Min(min int32) TimeType {
	t.min = min
	return t
}

func (TimeType) ID() int { return 43 } // minecraft:time

var timeUnitTicks = map[byte]float32{
	'd': 24000,
	's': 20,
	't': 1,
}

func (t TimeType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return nil, commander.NewParsingError(r, tc.Translatable(mcdata.ParsingFloatExpected))
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()

	unit := float32(1)
	numStr := raw
	if last := raw[len(raw)-1]; last >= 'a' && last <= 'z' {
		u, ok := timeUnitTicks[last]
		if !ok {
			r.SetCursor(start)
			return nil, commander.NewParsingError(r, tc.Translatable(mcdata.ArgumentTimeInvalidUnit))
		}
		unit = u
		numStr = raw[:len(raw)-1]
	}

	val, err := strconv.ParseFloat(numStr, 32)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingError(r, tc.Translatable(mcdata.ParsingFloatInvalid, tc.Text(numStr)))
	}

	ticks := int32(math.Round(float64(float32(val) * unit)))
	if ticks < t.min {
		r.SetCursor(start)
		return nil, commander.NewParsingError(r, tc.Translatable(
			mcdata.ArgumentTimeTickCountTooLow,
			tc.Text(strconv.Itoa(int(t.min))),
			tc.Text(strconv.Itoa(int(ticks))),
		))
	}
	return ticks, nil
}

func (t TimeType) WriteTo(w io.Writer) (int64, error) {
	return mc.Int(t.min).WriteTo(w)
}
