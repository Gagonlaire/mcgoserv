package commander

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
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

func (b BooleanType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	var val bool

	_, err := fmt.Fscan(r, &val)
	if err != nil {
		return nil, text_component.Text(err.Error()).SetColor("red")
	}
	return val, nil
}

func (b BooleanType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (f FloatType) Min(min float32) FloatType {
	f.min = min
	f.flags |= 0x01
	return f
}

func (f FloatType) Max(max float32) FloatType {
	f.max = max
	f.flags |= 0x02
	return f
}

func (f FloatType) ID() int { return 1 } // brigadier:float

func (f FloatType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	var val float32

	_, err := fmt.Fscan(r, &val)
	switch {
	case err != nil:
		return nil, text_component.Text(err.Error()).SetColor("red")
	case f.flags&0x01 != 0 && val < f.min:
		return nil, text_component.Text(fmt.Sprintf("float too small: must be >= %g", f.min)).SetColor("red")
	case f.flags&0x02 != 0 && val > f.max:
		return nil, text_component.Text(fmt.Sprintf("float too big: must be <= %g", f.max)).SetColor("red")
	default:
		return val, nil
	}
}

func (f FloatType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(f.flags).WriteTo(w)
	if f.flags&0x01 != 0 {
		nn, _ := mc.Float(f.min).WriteTo(w)
		n += nn
	}
	if f.flags&0x02 != 0 {
		nn, _ := mc.Float(f.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (d DoubleType) Min(min float64) DoubleType {
	d.min = min
	d.flags |= 0x01
	return d
}

func (d DoubleType) Max(max float64) DoubleType {
	d.max = max
	d.flags |= 0x02
	return d
}

func (d DoubleType) ID() int { return 2 } // brigadier:double

func (d DoubleType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	var val float64

	_, err := fmt.Fscan(r, &val)
	switch {
	case err != nil:
		return nil, text_component.Text(err.Error()).SetColor("red")
	case d.flags&0x01 != 0 && val < d.min:
		return nil, text_component.Text(fmt.Sprintf("double too small: must be >= %g", d.min)).SetColor("red")
	case d.flags&0x02 != 0 && val > d.max:
		return nil, text_component.Text(fmt.Sprintf("double too big: must be <= %g", d.max)).SetColor("red")
	default:
		return val, nil
	}
}

func (d DoubleType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(d.flags).WriteTo(w)
	if d.flags&0x01 != 0 {
		nn, _ := mc.Double(d.min).WriteTo(w)
		n += nn
	}
	if d.flags&0x02 != 0 {
		nn, _ := mc.Double(d.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (i IntType) Min(min int32) IntType {
	i.min = min
	i.flags |= 0x01
	return i
}

func (i IntType) Max(max int32) IntType {
	i.max = max
	i.flags |= 0x02
	return i
}

func (i IntType) ID() int { return 3 } // brigadier:integer

func (i IntType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	var val int32

	_, err := fmt.Fscan(r, &val)
	// todo: change error messages
	switch {
	case err != nil:
		return nil, text_component.Text(err.Error()).SetColor("red")
	case i.flags&0x01 != 0 && val < i.min:
		return nil, text_component.Text(fmt.Sprintf("integer too small: must be >= %d", i.min)).SetColor("red")
	case i.flags&0x02 != 0 && val > i.max:
		return nil, text_component.Text(fmt.Sprintf("integer too big: must be <= %d", i.max)).SetColor("red")
	default:
		return val, nil
	}
}

func (i IntType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.Byte(i.flags).WriteTo(w)
	if i.flags&0x01 != 0 {
		nn, _ := mc.Int(i.min).WriteTo(w)
		n += nn
	}
	if i.flags&0x02 != 0 {
		nn, _ := mc.Int(i.max).WriteTo(w)
		n += nn
	}
	return n, nil
}

func (l LongType) Min(min int64) LongType {
	l.min = min
	l.flags |= 0x01
	return l
}

func (l LongType) Max(max int64) LongType {
	l.max = max
	l.flags |= 0x02
	return l
}

func (l LongType) ID() int { return 4 } // brigadier:long

func (l LongType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	var val int64

	_, err := fmt.Fscan(r, &val)
	switch {
	case err != nil:
		return nil, text_component.Text(err.Error()).SetColor("red")
	case l.flags&0x01 != 0 && val < l.min:
		return nil, text_component.Text(fmt.Sprintf("long too small: must be >= %d", l.min)).SetColor("red")
	case l.flags&0x02 != 0 && val > l.max:
		return nil, text_component.Text(fmt.Sprintf("long too big: must be <= %d", l.max)).SetColor("red")
	default:
		return val, nil
	}
}

func (l LongType) WriteTo(w io.Writer) (int64, error) {
	n, _ := mc.UnsignedByte(l.flags).WriteTo(w)
	if l.flags&0x01 != 0 {
		nn, _ := mc.Long(l.min).WriteTo(w)
		n += nn
	}
	if l.flags&0x02 != 0 {
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

func (s StringType) Parse(r *strings.Reader) (interface{}, text_component.Component) {
	switch s.behavior {
	case GreedyPhrase:
		var b strings.Builder
		for r.Len() > 0 {
			ch, _, _ := r.ReadRune()
			b.WriteRune(ch)
		}
		val := b.String()
		if len(val) == 0 {
			return nil, text_component.Text("expected a string").SetColor("red")
		}
		return val, nil
	case QuotablePhrase:
		ch, _, err := r.ReadRune()
		if err != nil {
			return nil, text_component.Text("expected a string").SetColor("red")
		}
		if ch == '"' {
			var b strings.Builder
			for {
				c, _, err := r.ReadRune()
				if err != nil {
					return nil, text_component.Text("unclosed quoted string").SetColor("red")
				}
				if c == '\\' {
					next, _, err := r.ReadRune()
					if err != nil {
						return nil, text_component.Text("unexpected end of escape sequence").SetColor("red")
					}
					b.WriteRune(next)
					continue
				}
				if c == '"' {
					return b.String(), nil
				}
				b.WriteRune(c)
			}
		}
		_ = r.UnreadRune()
		fallthrough
	case SingleWord:
		var b strings.Builder
		for r.Len() > 0 {
			ch, _, _ := r.ReadRune()
			if ch == ' ' {
				_ = r.UnreadRune()
				break
			}
			b.WriteRune(ch)
		}
		val := b.String()
		if len(val) == 0 {
			return nil, text_component.Text("expected a string").SetColor("red")
		}
		return val, nil
	default:
		panic("invalid string behavior")
	}
}

func (s StringType) WriteTo(w io.Writer) (int64, error) {
	return mc.VarInt(s.behavior).WriteTo(w)
}
