package nbtpath

import "errors"

var (
	ErrPathNotFound     = errors.New("nbt: path not found")
	ErrNotAList         = errors.New("nbt: value is not a list")
	ErrNotACompound     = errors.New("nbt: value is not a compound")
	ErrIndexOOB         = errors.New("nbt: index out of bounds")
	ErrEmptyPath        = errors.New("nbt: empty path")
	ErrExpectedCompound = errors.New("nbt: expected compound")
	ErrSelectsRoot      = errors.New("nbt: path selects root")
	ErrNotAString       = errors.New("nbt: value is not a string")
	ErrMultipleValues   = errors.New("nbt: path resolves to multiple values")
	ErrStringBounds     = errors.New("nbt: substring bounds out of range")
)
