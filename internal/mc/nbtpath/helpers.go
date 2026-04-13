package nbtpath

import (
	"fmt"
	"reflect"
	"strings"
)

func navigateParent(data any, nodes []Node) (any, Node, error) {
	if len(nodes) == 0 {
		return nil, Node{}, ErrEmptyPath
	}
	if len(nodes) == 1 {
		return data, nodes[0], nil
	}
	parent, err := Navigate(data, nodes[:len(nodes)-1])
	if err != nil {
		return nil, Node{}, err
	}
	return parent, nodes[len(nodes)-1], nil
}

func setAtNode(parent any, node Node, value any) error {
	if node.Name != "" {
		m, ok := parent.(map[string]any)
		if !ok {
			return ErrPathNotFound
		}
		m[node.Name] = value
		return nil
	}
	return ErrPathNotFound
}

func getList(data any, nodes []Node) (any, Node, []any, error) {
	parent, lastNode, err := navigateParent(data, nodes)
	if err != nil {
		return nil, Node{}, nil, err
	}

	var target any
	if lastNode.Name != "" {
		m, ok := parent.(map[string]any)
		if !ok {
			return nil, Node{}, nil, ErrPathNotFound
		}
		t, exists := m[lastNode.Name]
		if !exists {
			return nil, Node{}, nil, ErrPathNotFound
		}
		target = t
	} else {
		target = parent
	}

	list, ok := toAnySlice(target)
	if !ok {
		return nil, Node{}, nil, ErrNotAList
	}
	return parent, lastNode, list, nil
}

func toAnySlice(v any) ([]any, bool) {
	if s, ok := v.([]any); ok {
		return s, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}
	result := make([]any, rv.Len())
	for i := 0; i < rv.Len(); i++ {
		result[i] = rv.Index(i).Interface()
	}
	return result, true
}

func compoundMatch(candidate, filter any) bool {
	cm, ok1 := candidate.(map[string]any)
	fm, ok2 := filter.(map[string]any)
	if !ok1 || !ok2 {
		return false
	}
	for k, fv := range fm {
		cv, exists := cm[k]
		if !exists {
			return false
		}
		if fMap, ok := fv.(map[string]any); ok {
			if !compoundMatch(cv, fMap) {
				return false
			}
			continue
		}
		if !reflect.DeepEqual(cv, fv) {
			return false
		}
	}
	return true
}

// todo: update format to match Minecraft style and support text components
func writeSNBT(sb *strings.Builder, v any) {
	switch val := v.(type) {
	case map[string]any:
		sb.WriteByte('{')
		first := true
		for k, v := range val {
			if !first {
				sb.WriteByte(',')
			}
			first = false
			writeSNBTKey(sb, k)
			sb.WriteByte(':')
			writeSNBT(sb, v)
		}
		sb.WriteByte('}')

	case []any:
		sb.WriteByte('[')
		for i, elem := range val {
			if i > 0 {
				sb.WriteByte(',')
			}
			writeSNBT(sb, elem)
		}
		sb.WriteByte(']')

	case []byte:
		sb.WriteString("[B;")
		for i, b := range val {
			if i > 0 {
				sb.WriteByte(',')
			}
			_, _ = fmt.Fprintf(sb, "%dB", int8(b))
		}
		sb.WriteByte(']')

	case []int32:
		sb.WriteString("[I;")
		for i, n := range val {
			if i > 0 {
				sb.WriteByte(',')
			}
			_, _ = fmt.Fprintf(sb, "%d", n)
		}
		sb.WriteByte(']')

	case []int64:
		sb.WriteString("[L;")
		for i, n := range val {
			if i > 0 {
				sb.WriteByte(',')
			}
			_, _ = fmt.Fprintf(sb, "%dL", n)
		}
		sb.WriteByte(']')

	case int8:
		_, _ = fmt.Fprintf(sb, "%dB", val)
	case int16:
		_, _ = fmt.Fprintf(sb, "%dS", val)
	case int32:
		_, _ = fmt.Fprintf(sb, "%d", val)
	case int64:
		_, _ = fmt.Fprintf(sb, "%dL", val)
	case float32:
		_, _ = fmt.Fprintf(sb, "%gF", val)
	case float64:
		_, _ = fmt.Fprintf(sb, "%gD", val)
	case string:
		writeSNBTString(sb, val)
	case bool:
		if val {
			sb.WriteString("1B")
		} else {
			sb.WriteString("0B")
		}
	default:
		_, _ = fmt.Fprintf(sb, "%v", val)
	}
}

func writeSNBTKey(sb *strings.Builder, key string) {
	safe := true
	for _, c := range []byte(key) {
		if !isUnquotedChar(c) {
			safe = false
			break
		}
	}
	if safe && key != "" {
		sb.WriteString(key)
	} else {
		writeSNBTString(sb, key)
	}
}

func writeSNBTString(sb *strings.Builder, s string) {
	sb.WriteByte('"')
	for _, c := range s {
		if c == '"' || c == '\\' {
			sb.WriteByte('\\')
		}
		sb.WriteRune(c)
	}
	sb.WriteByte('"')
}

func isUnquotedChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '_' || c == '-' || c == '.' || c == '+'
}
