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

type BooleanType struct{}

type FloatType struct {
	min   float32
	max   float32
	flags byte
}

type DoubleType struct {
	min   float64
	max   float64
	flags byte
}

type IntType struct {
	min   int32
	max   int32
	flags byte
}

type LongType struct {
	min   int64
	max   int64
	flags byte
}

type StringBehavior int

type StringType struct {
	behavior StringBehavior
}

const (
	FlagMin = 0x01
	FlagMax = 0x02
)

const (
	SingleWord StringBehavior = iota
	QuotablePhrase
	GreedyPhrase
)

var Boolean = BooleanType{}

var Float = FloatType{
	min: -math.MaxFloat32,
	max: math.MaxFloat32,
}

var Double = DoubleType{
	min: -math.MaxFloat64,
	max: math.MaxFloat64,
}

var Int = IntType{
	min: math.MinInt32,
	max: math.MaxInt32,
}

var Long = LongType{
	min: math.MinInt64,
	max: math.MaxInt64,
}

var String = StringType{
	behavior: SingleWord,
}

func (b BooleanType) ID() int { return 0 } // brigadier:bool

func (b BooleanType) Parse(r *commander.CommandReader) (any, error) {
	// to match brigadier's behavior with primitive types
	if !r.CanRead() || !commander.IsAllowedInUnquotedString(r.Peek()) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingBoolExpected), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingBoolInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
}

func (b BooleanType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (f FloatType) Min(min float32) FloatType {
	f.min = min
	f.flags |= FlagMin
	return f
}

func (f FloatType) Max(max float32) FloatType {
	f.max = max
	f.flags |= FlagMax
	return f
}

func (f FloatType) ID() int { return 1 } // brigadier:float

func (f FloatType) Parse(r *commander.CommandReader) (any, error) {
	// to match brigadier's behavior with primitive types
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingFloatExpected), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()
	val, err := strconv.ParseFloat(raw, 32)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingFloatInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	v := float32(val)
	if f.flags&FlagMin != 0 && v < f.min {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentFloatLow,
				tc.Text(strconv.FormatFloat(float64(f.min), 'f', -1, 32)),
				tc.Text(strconv.FormatFloat(float64(v), 'f', -1, 32)),
			),
			r.Input(),
		)
	}
	if f.flags&FlagMax != 0 && v > f.max {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentFloatBig,
				tc.Text(strconv.FormatFloat(float64(f.max), 'f', -1, 32)),
				tc.Text(strconv.FormatFloat(float64(v), 'f', -1, 32)),
			),
			r.Input(),
		)
	}
	return v, nil
}

func (f FloatType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(f.flags).WriteTo(w)
	if f.flags&FlagMin != 0 {
		nn, _ := mc.Float(f.min).WriteTo(w)
		n += nn
	}
	if f.flags&FlagMax != 0 {
		nn, _ := mc.Float(f.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (d DoubleType) Min(min float64) DoubleType {
	d.min = min
	d.flags |= FlagMin
	return d
}

func (d DoubleType) Max(max float64) DoubleType {
	d.max = max
	d.flags |= FlagMax
	return d
}

func (d DoubleType) ID() int { return 2 } // brigadier:double

func (d DoubleType) Parse(r *commander.CommandReader) (any, error) {
	// to match brigadier's behavior with primitive types
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingDoubleExpected), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingDoubleInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	if d.flags&FlagMin != 0 && val < d.min {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentDoubleLow,
				tc.Text(strconv.FormatFloat(d.min, 'f', -1, 64)),
				tc.Text(strconv.FormatFloat(val, 'f', -1, 64)),
			),
			r.Input(),
		)
	}
	if d.flags&FlagMax != 0 && val > d.max {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentDoubleBig,
				tc.Text(strconv.FormatFloat(d.max, 'f', -1, 64)),
				tc.Text(strconv.FormatFloat(val, 'f', -1, 64)),
			),
			r.Input(),
		)
	}
	return val, nil
}

func (d DoubleType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(d.flags).WriteTo(w)
	if d.flags&FlagMin != 0 {
		nn, _ := mc.Double(d.min).WriteTo(w)
		n += nn
	}
	if d.flags&FlagMax != 0 {
		nn, _ := mc.Double(d.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (i IntType) Min(min int32) IntType {
	i.min = min
	i.flags |= FlagMin
	return i
}

func (i IntType) Max(max int32) IntType {
	i.max = max
	i.flags |= FlagMax
	return i
}

func (i IntType) ID() int { return 3 } // brigadier:integer

func (i IntType) Parse(r *commander.CommandReader) (any, error) {
	// to match brigadier's behavior with primitive types
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingIntExpected), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()
	val, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingIntInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	v := int32(val)
	if i.flags&FlagMin != 0 && v < i.min {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentIntegerLow,
				tc.Text(strconv.Itoa(int(i.min))),
				tc.Text(strconv.Itoa(int(v))),
			),
			r.Input(),
		)
	}
	if i.flags&FlagMax != 0 && v > i.max {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentIntegerBig,
				tc.Text(strconv.Itoa(int(i.max))),
				tc.Text(strconv.Itoa(int(v))),
			),
			r.Input(),
		)
	}
	return v, nil
}

func (i IntType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(i.flags).WriteTo(w)
	if i.flags&FlagMin != 0 {
		nn, _ := mc.Int(i.min).WriteTo(w)
		n += nn
	}
	if i.flags&FlagMax != 0 {
		nn, _ := mc.Int(i.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (l LongType) Min(min int64) LongType {
	l.min = min
	l.flags |= FlagMin
	return l
}

func (l LongType) Max(max int64) LongType {
	l.max = max
	l.flags |= FlagMax
	return l
}

func (l LongType) ID() int { return 4 } // brigadier:long

func (l LongType) Parse(r *commander.CommandReader) (any, error) {
	// to match brigadier's behavior with primitive types
	if !r.CanRead() || !commander.IsAllowedInNumericUnquotedString(r.Peek()) {
		return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingLongExpected), r.Input(), r.Cursor())
	}

	start := r.Cursor()
	raw := r.ReadUnquotedString()
	val, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ParsingLongInvalid, tc.Text(raw)),
			r.Input(), start,
		)
	}
	if l.flags&FlagMin != 0 && val < l.min {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentLongLow,
				tc.Text(strconv.FormatInt(l.min, 10)),
				tc.Text(strconv.FormatInt(val, 10)),
			),
			r.Input(),
		)
	}
	if l.flags&FlagMax != 0 && val > l.max {
		r.SetCursor(start)
		return nil, commander.NewParsingError(
			tc.Translatable(
				mcdata.ArgumentLongBig,
				tc.Text(strconv.FormatInt(l.max, 10)),
				tc.Text(strconv.FormatInt(val, 10)),
			),
			r.Input(),
		)
	}
	return val, nil
}

func (l LongType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(l.flags).WriteTo(w)
	if l.flags&FlagMin != 0 {
		nn, _ := mc.Long(l.min).WriteTo(w)
		n += nn
	}
	if l.flags&FlagMax != 0 {
		nn, _ := mc.Long(l.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (s StringType) Behavior(behavior StringBehavior) StringType {
	s.behavior = behavior
	return s
}

func (s StringType) ID() int { return 5 } // brigadier:string

func (s StringType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	switch s.behavior {
	case GreedyPhrase:
		remaining := r.GetRemaining()
		r.SetCursor(r.TotalLength())
		return remaining, nil
	case QuotablePhrase:
		val, err := r.ReadString()
		if err != nil {
			return nil, err
		}
		if len(val) == 0 && r.Cursor() == start {
			return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingQuoteExpectedStart), r.Input(), start)
		}
		return val, nil
	case SingleWord:
		val := r.ReadUnquotedString()
		if len(val) == 0 {
			return nil, commander.NewParsingErrorAt(tc.Translatable(mcdata.ParsingQuoteExpectedStart), r.Input(), start)
		}
		return val, nil
	default:
		panic("commander: invalid string behavior")
	}
}

func (s StringType) WriteTo(w io.Writer) (int64, error) {
	return mc.VarInt(s.behavior).WriteTo(w)
}
