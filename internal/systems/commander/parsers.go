package commander

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

type IntType struct {
	MinVal int
	MaxVal int
	hasMin bool
	hasMax bool
}

func (i IntType) WriteTo(w io.Writer) (int64, error) {
	flags := byte(0)

	if i.hasMin {
		flags |= 0x01
	}
	if i.hasMax {
		flags |= 0x02
	}
	n, _ := mc.Byte(flags).WriteTo(w)
	if i.hasMin {
		nn, _ := mc.Int(i.MinVal).WriteTo(w)
		n += nn
	}
	if i.hasMax {
		nn, _ := mc.Int(i.MaxVal).WriteTo(w)
		n += nn
	}
	return n, nil
}

var Int = IntType{
	MinVal: math.MinInt32,
	MaxVal: math.MaxInt32,
}

func (i IntType) Min(min int) IntType {
	i.MinVal = min
	i.hasMin = true
	return i
}

func (i IntType) Max(max int) IntType {
	i.MaxVal = max
	i.hasMax = true
	return i
}

func (i IntType) ID() int { return 3 }

// todo: parser should maybe return a ErrorCode instead
// todo: parser other than brigadier should maybe be outside of this package
func (i IntType) Parse(r *strings.Reader) (interface{}, error) {
	var val int

	_, err := fmt.Fscan(r, &val)
	switch {
	case err != nil:
		return nil, err
	case i.hasMin && val < i.MinVal:
		return nil, fmt.Errorf("integer too small: must be >= %d", i.MinVal)
	case i.hasMax && val > i.MaxVal:
		return nil, fmt.Errorf("integer too big: must be <= %d", i.MaxVal)
	default:
		return val, nil
	}
}
