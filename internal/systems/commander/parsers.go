package commander

import (
	"fmt"
	"math"
	"strings"
)

type IntType struct {
	MinVal int
	MaxVal int
	hasMin bool
	hasMax bool
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

func (i IntType) ID() string { return "brigadier:integer" }

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
