package nbtpath

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// WriteSNBT walks v and emits typed segments to sink.
// NOTE: compounds are sorted for deterministic output
func WriteSNBT(sink Sink, v any) {
	switch val := v.(type) {
	case map[string]any:
		sink.Punct("{")
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				sink.Punct(", ")
			}
			writeKey(sink, k)
			sink.Punct(": ")
			WriteSNBT(sink, val[k])
		}
		sink.Punct("}")

	case []any:
		sink.Punct("[")
		for i, elem := range val {
			if i > 0 {
				sink.Punct(", ")
			}
			WriteSNBT(sink, elem)
		}
		sink.Punct("]")

	case []byte:
		writeArrayHeader(sink, "B")
		for i, b := range val {
			if i > 0 {
				sink.Punct(", ")
			}
			sink.Number(strconv.FormatInt(int64(int8(b)), 10))
			sink.Type("b")
		}
		sink.Punct("]")

	case []int32:
		writeArrayHeader(sink, "I")
		for i, n := range val {
			if i > 0 {
				sink.Punct(", ")
			}
			sink.Number(strconv.FormatInt(int64(n), 10))
		}
		sink.Punct("]")

	case []int64:
		writeArrayHeader(sink, "L")
		for i, n := range val {
			if i > 0 {
				sink.Punct(", ")
			}
			sink.Number(strconv.FormatInt(n, 10))
			sink.Type("l")
		}
		sink.Punct("]")

	case int8:
		sink.Number(strconv.FormatInt(int64(val), 10))
		sink.Type("b")
	case int16:
		sink.Number(strconv.FormatInt(int64(val), 10))
		sink.Type("s")
	case int32:
		sink.Number(strconv.FormatInt(int64(val), 10))
	case int64:
		sink.Number(strconv.FormatInt(val, 10))
		sink.Type("l")
	case float32:
		sink.Number(formatFloat(float64(val), 32))
		sink.Type("f")
	case float64:
		sink.Number(formatFloat(val, 64))
		sink.Type("d")
	case string:
		writeString(sink, val)
	case bool:
		if val {
			sink.Number("1")
		} else {
			sink.Number("0")
		}
		sink.Type("b")
	default:
		sink.Number(fmt.Sprintf("%v", val))
	}
}

func writeArrayHeader(sink Sink, typeLetter string) {
	sink.Punct("[")
	sink.Type(typeLetter)
	sink.Punct("; ")
}

func writeKey(sink Sink, key string) {
	safe := key != ""
	for _, b := range []byte(key) {
		if !isUnquotedChar(b) {
			safe = false
			break
		}
	}
	if safe {
		sink.Key(key)
		return
	}
	writeString(sink, key)
}

func writeString(sink Sink, s string) {
	var inner strings.Builder
	for _, r := range s {
		if r == '"' || r == '\\' {
			inner.WriteByte('\\')
		}
		inner.WriteRune(r)
	}
	sink.Punct(`"`)
	sink.String(inner.String())
	sink.Punct(`"`)
}

func formatFloat(v float64, bits int) string {
	s := strconv.FormatFloat(v, 'g', -1, bits)
	if !strings.ContainsAny(s, ".eEnN") {
		s += ".0"
	}
	return s
}

func isUnquotedChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == '+'
}
