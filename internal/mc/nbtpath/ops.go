package nbtpath

import (
	"bytes"
	"errors"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// NbtReader read nbt access
type NbtReader interface {
	NbtData() (nbt.StringifiedMessage, error)
	NbtGet(path *Path) (nbt.StringifiedMessage, error)
}

// NbtAccessor read/write nbt access
// todo: also split Writer in an interface
type NbtAccessor interface {
	NbtReader
	NbtMerge(compound nbt.StringifiedMessage) error
	NbtAppend(path *Path, value nbt.StringifiedMessage) error
	NbtPrepend(path *Path, value nbt.StringifiedMessage) error
	NbtInsert(path *Path, index int, value nbt.StringifiedMessage) error
	NbtRemove(path *Path) error
}

// todo: maybe replace with the mc errors directly
var (
	ErrPathNotFound = errors.New("nbt: path not found")
	ErrNotAList     = errors.New("nbt: value is not a list")
	ErrIndexOOB     = errors.New("nbt: index out of bounds")
	ErrEmptyPath    = errors.New("nbt: empty path")
)

func ToMap(v any) (map[string]any, error) {
	bin, err := nbt.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := nbt.Unmarshal(bin, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = make(map[string]any)
	}
	return m, nil
}

func MapToSNBT(m map[string]any) (nbt.StringifiedMessage, error) {
	var value any
	bin, err := nbt.Marshal(m)
	if err != nil {
		return "", err
	}
	if err := nbt.Unmarshal(bin, &value); err != nil {
		return "", err
	}
	return FormatSNBT(value), nil
}

func ValueToSNBT(v any) (nbt.StringifiedMessage, error) {
	bin, err := nbt.Marshal(v)
	if err != nil {
		return "", err
	}
	var value any
	if err := nbt.Unmarshal(bin, &value); err != nil {
		return "", err
	}
	return FormatSNBT(value), nil
}

func SNBTToValue(snbt nbt.StringifiedMessage) (any, error) {
	var buf bytes.Buffer
	tagType := snbt.TagType()
	if tagType == nbt.TagEnd {
		return nil, &nbt.SyntaxError{Message: "invalid SNBT"}
	}

	// write root tag header = tagType + empty name (2 bytes len=0)
	buf.WriteByte(tagType)
	buf.WriteByte(0)
	buf.WriteByte(0)
	if err := snbt.MarshalNBT(&buf); err != nil {
		return nil, err
	}

	var value any
	if err := nbt.Unmarshal(buf.Bytes(), &value); err != nil {
		return nil, err
	}
	return value, nil
}

// Navigate walks a parsed NBT path through nested maps/slices.
func Navigate(data any, nodes []Node) (any, error) {
	current := data
	for _, node := range nodes {
		switch {
		case node.Name != "":
			m, ok := current.(map[string]any)
			if !ok {
				return nil, ErrPathNotFound
			}
			val, exists := m[node.Name]
			if !exists {
				return nil, ErrPathNotFound
			}
			current = val

		case node.Index >= 0:
			list, ok := toAnySlice(current)
			if !ok {
				return nil, ErrNotAList
			}
			if node.Index >= len(list) {
				return nil, ErrIndexOOB
			}
			current = list[node.Index]

		case node.IsMatch && node.Filter != "":
			list, ok := toAnySlice(current)
			if !ok {
				return nil, ErrNotAList
			}
			filterVal, err := SNBTToValue(node.Filter)
			if err != nil {
				return nil, err
			}
			found := false
			for _, elem := range list {
				if compoundMatch(elem, filterVal) {
					current = elem
					found = true
					break
				}
			}
			if !found {
				return nil, ErrPathNotFound
			}

		case node.IsMatch:
		default:
			// for both cases, return all elements
		}
	}
	return current, nil
}

func MergeOnto(dst any, compound nbt.StringifiedMessage) error {
	value, err := SNBTToValue(compound)
	if err != nil {
		return err
	}
	if _, ok := value.(map[string]any); !ok {
		return errors.New("nbt: merge source must be a compound")
	}
	bin, err := nbt.Marshal(value)
	if err != nil {
		return err
	}
	return nbt.Unmarshal(bin, dst)
}

func WriteBack(dst any, m map[string]any) error {
	bin, err := nbt.Marshal(m)
	if err != nil {
		return err
	}
	return nbt.Unmarshal(bin, dst)
}

func ListAppend(data map[string]any, nodes []Node, value any) error {
	parent, node, list, err := getList(data, nodes)
	if err != nil {
		return err
	}
	return setAtNode(parent, node, append(list, value))
}

func ListPrepend(data map[string]any, nodes []Node, value any) error {
	parent, node, list, err := getList(data, nodes)
	if err != nil {
		return err
	}
	return setAtNode(parent, node, append([]any{value}, list...))
}

func ListInsert(data map[string]any, nodes []Node, index int, value any) error {
	parent, node, list, err := getList(data, nodes)
	if err != nil {
		return err
	}
	if index < 0 || index > len(list) {
		return ErrIndexOOB
	}
	result := make([]any, 0, len(list)+1)
	result = append(result, list[:index]...)
	result = append(result, value)
	result = append(result, list[index:]...)
	return setAtNode(parent, node, result)
}

func RemoveAtPath(data map[string]any, nodes []Node) error {
	parent, lastNode, err := navigateParent(data, nodes)
	if err != nil {
		return err
	}

	if lastNode.Name != "" {
		m, ok := parent.(map[string]any)
		if !ok {
			return ErrPathNotFound
		}
		delete(m, lastNode.Name)
		return nil
	}

	if lastNode.Index >= 0 {
		list, ok := toAnySlice(parent)
		if !ok {
			return ErrNotAList
		}
		if lastNode.Index >= len(list) {
			return ErrIndexOOB
		}
		if len(nodes) < 2 {
			return ErrPathNotFound
		}
		grandparent, parentNode, err := navigateParent(data, nodes[:len(nodes)-1])
		if err != nil {
			return err
		}
		newList := append(list[:lastNode.Index], list[lastNode.Index+1:]...)
		return setAtNode(grandparent, parentNode, newList)
	}

	return ErrPathNotFound
}

func FormatSNBT(v any) nbt.StringifiedMessage {
	var sb strings.Builder
	writeSNBT(&sb, v)
	return nbt.StringifiedMessage(sb.String())
}
