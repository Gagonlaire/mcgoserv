package parsers

import (
	"io"
	"strconv"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Tnze/go-mc/nbt"
)

type NbtCompoundTagType struct{}

var NbtCompoundTag = NbtCompoundTagType{}

func (n NbtCompoundTagType) ID() int { return 21 } // minecraft:nbt_compound_tag

func (n NbtCompoundTagType) Parse(r *commander.CommandReader) (nbt.StringifiedMessage, error) {
	start := r.Cursor()
	raw, err := readSNBT(r)
	if err != nil {
		return "", err
	}

	// todo: find a way to avoid the double parsing
	tagType, err := validateSNBT(raw)
	if err != nil {
		r.SetCursor(start)
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}

	if tagType != nbt.TagCompound {
		r.SetCursor(start)
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedCompound),
			r.Input(), start,
		)
	}

	return nbt.StringifiedMessage(raw), nil
}

func (n NbtCompoundTagType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

type NbtTagType struct{}

var NbtTag = NbtTagType{}

func (n NbtTagType) ID() int { return 22 } // minecraft:nbt_tag

func (n NbtTagType) Parse(r *commander.CommandReader) (nbt.StringifiedMessage, error) {
	start := r.Cursor()
	raw, err := readSNBT(r)
	if err != nil {
		return "", err
	}

	_, err = validateSNBT(raw)
	if err != nil {
		r.SetCursor(start)
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}

	return nbt.StringifiedMessage(raw), nil
}

func (n NbtTagType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

var NbtPath = NbtPathType{}

type NbtPathType struct{}

type NbtPathNode struct {
	Name    string
	Index   int
	IsMatch bool
	Filter  nbt.StringifiedMessage
}

type ParsedNbtPath struct {
	Nodes []NbtPathNode
	Raw   string
}

func (n NbtPathType) ID() int { return 23 } // minecraft:nbt_path

func (n NbtPathType) Parse(r *commander.CommandReader) (*ParsedNbtPath, error) {
	start := r.Cursor()
	var nodes []NbtPathNode

	for r.CanRead() && r.Peek() != ' ' {
		parsed, err := readNbtPathNode(r)
		if err != nil {
			r.SetCursor(start)
			return nil, err
		}
		nodes = append(nodes, parsed...)

		if r.CanRead() && r.Peek() == '.' {
			r.Skip()
		}
	}
	if len(nodes) == 0 {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}

	return &ParsedNbtPath{
		Nodes: nodes,
		Raw:   r.Input()[start:r.Cursor()],
	}, nil
}

func (n NbtPathType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func readSNBT(r *commander.CommandReader) (string, error) {
	if !r.CanRead() {
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}

	start := r.Cursor()
	ch := r.Peek()

	switch ch {
	case '{', '[':
		if err := readSNBTBalanced(r); err != nil {
			return "", err
		}
	case '"', '\'':
		if err := readSNBTQuoted(r); err != nil {
			return "", err
		}
	default:
		r.ReadUnquotedString()
		if r.Cursor() == start {
			return "", commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentNbtExpectedValue),
				r.Input(), r.Cursor(),
			)
		}
	}

	return r.Input()[start:r.Cursor()], nil
}

func readSNBTBalanced(r *commander.CommandReader) error {
	depth := 0
	for r.CanRead() {
		ch := r.Peek()
		switch ch {
		case '{', '[':
			depth++
			r.Skip()
		case '}', ']':
			depth--
			r.Skip()
			if depth == 0 {
				return nil
			}
		case '"', '\'':
			if err := readSNBTQuoted(r); err != nil {
				return err
			}
		default:
			r.Skip()
		}
	}

	return commander.NewParsingErrorAt(
		tc.Translatable(mcdata.ArgumentNbtExpectedValue),
		r.Input(), r.Cursor(),
	)
}

func readSNBTQuoted(r *commander.CommandReader) error {
	quote := r.Read()
	for r.CanRead() {
		ch := r.Read()
		if ch == '\\' {
			if !r.CanRead() {
				return commander.NewParsingErrorAt(
					tc.Translatable(mcdata.ArgumentNbtExpectedValue),
					r.Input(), r.Cursor(),
				)
			}
			r.Skip()
		} else if ch == quote {
			return nil
		}
	}

	return commander.NewParsingErrorAt(
		tc.Translatable(mcdata.ArgumentNbtExpectedValue),
		r.Input(), r.Cursor(),
	)
}

func validateSNBT(raw string) (byte, error) {
	msg := nbt.StringifiedMessage(raw)
	tagType := msg.TagType()
	if tagType == nbt.TagEnd {
		return 0, &nbt.SyntaxError{Message: "invalid SNBT"}
	}
	return tagType, nil
}

func readNbtPathNode(r *commander.CommandReader) ([]NbtPathNode, error) {
	if !r.CanRead() || r.Peek() == ' ' {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}

	ch := r.Peek()
	if ch == '{' {
		filter, err := readCompoundFilter(r)
		if err != nil {
			return nil, err
		}
		return []NbtPathNode{{IsMatch: true, Index: -1, Filter: filter}}, nil
	}
	if ch == '[' {
		node, err := readNbtPathIndex(r)
		if err != nil {
			return nil, err
		}
		return []NbtPathNode{node}, nil
	}
	name, err := readNbtPathKey(r)
	if err != nil {
		return nil, err
	}
	node := NbtPathNode{Name: name, Index: -1}
	var extra []NbtPathNode
	for r.CanRead() {
		c := r.Peek()
		if c == '{' && !node.IsMatch {
			filter, err := readCompoundFilter(r)
			if err != nil {
				return nil, err
			}
			node.IsMatch = true
			node.Filter = filter
		} else if c == '[' {
			idxNode, err := readNbtPathIndex(r)
			if err != nil {
				return nil, err
			}
			extra = append(extra, idxNode)
		} else {
			break
		}
	}

	return append([]NbtPathNode{node}, extra...), nil
}

func readNbtPathKey(r *commander.CommandReader) (string, error) {
	if !r.CanRead() {
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedKey),
			r.Input(), r.Cursor(),
		)
	}
	ch := r.Peek()
	if ch == '"' || ch == '\'' {
		val, err := r.ReadQuotedString()
		if err != nil {
			return "", err
		}
		return val, nil
	}
	start := r.Cursor()
	for r.CanRead() {
		c := r.Peek()
		if c == '.' || c == '[' || c == '{' || c == ' ' {
			break
		}
		r.Skip()
	}
	if r.Cursor() == start {
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedKey),
			r.Input(), r.Cursor(),
		)
	}

	return r.Input()[start:r.Cursor()], nil
}

func readNbtPathIndex(r *commander.CommandReader) (NbtPathNode, error) {
	r.Skip()
	if !r.CanRead() {
		return NbtPathNode{}, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	ch := r.Peek()
	if ch == ']' {
		r.Skip()
		return NbtPathNode{Index: -1}, nil
	}
	if ch == '{' {
		filter, err := readCompoundFilter(r)
		if err != nil {
			return NbtPathNode{}, err
		}
		if !r.CanRead() || r.Peek() != ']' {
			return NbtPathNode{}, commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentNbtExpectedValue),
				r.Input(), r.Cursor(),
			)
		}
		r.Skip()
		return NbtPathNode{Index: -1, IsMatch: true, Filter: filter}, nil
	}
	start := r.Cursor()
	if r.CanRead() && r.Peek() == '-' {
		r.Skip()
	}
	for r.CanRead() && r.Peek() >= '0' && r.Peek() <= '9' {
		r.Skip()
	}
	if r.Cursor() == start {
		return NbtPathNode{}, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	idx, err := strconv.Atoi(r.Input()[start:r.Cursor()])
	if err != nil {
		return NbtPathNode{}, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}
	if !r.CanRead() || r.Peek() != ']' {
		return NbtPathNode{}, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	r.Skip()

	return NbtPathNode{Index: idx}, nil
}

func readCompoundFilter(r *commander.CommandReader) (nbt.StringifiedMessage, error) {
	start := r.Cursor()
	if err := readSNBTBalanced(r); err != nil {
		return "", err
	}
	raw := r.Input()[start:r.Cursor()]
	tagType, err := validateSNBT(raw)
	if err != nil || tagType != nbt.TagCompound {
		return "", commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedCompound),
			r.Input(), start,
		)
	}

	return nbt.StringifiedMessage(raw), nil
}
