package commander

import (
	"fmt"
)

func GetArgument[T any](args ParsedArgs, name string) T {
	val, ok := args[name]
	if !ok {
		panic(fmt.Errorf("commander: argument '%s' not found", name))
	}
	tVal, ok := val.(T)
	if !ok {
		var zero T
		panic(fmt.Errorf("commander: argument '%s' is not of type %T, got %T", name, zero, val))
	}
	return tVal
}

func (p ParsedArgs) GetBool(name string) bool {
	return GetArgument[bool](p, name)
}

func (p ParsedArgs) GetFloat(name string) float32 {
	return GetArgument[float32](p, name)
}

func (p ParsedArgs) GetDouble(name string) float64 {
	return GetArgument[float64](p, name)
}

func (p ParsedArgs) GetInt(name string) int32 {
	return GetArgument[int32](p, name)
}

func (p ParsedArgs) GetLong(name string) int64 {
	return GetArgument[int64](p, name)
}

func (p ParsedArgs) GetString(name string) string {
	return GetArgument[string](p, name)
}
