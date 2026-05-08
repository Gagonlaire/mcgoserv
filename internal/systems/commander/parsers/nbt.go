package parsers

import (
	"io"
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Tnze/go-mc/nbt"
)

type NbtCompoundTagType struct{}

var NbtCompoundTag = NbtCompoundTagType{}

func (n NbtCompoundTagType) ID() int { return 21 } // minecraft:nbt_compound_tag

func (n NbtCompoundTagType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	raw, err := readSNBT(r)
	if err != nil {
		return nil, err
	}

	// todo: find a way to avoid the double parsing
	tagType, err := validateSNBT(raw)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}

	if tagType != nbt.TagCompound {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
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

func (n NbtTagType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	raw, err := readSNBT(r)
	if err != nil {
		return nil, err
	}

	_, err = validateSNBT(raw)
	if err != nil {
		r.SetCursor(start)
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}

	return nbt.StringifiedMessage(raw), nil
}

func (n NbtTagType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

var NbtPath = NbtPathType{}

type NbtPathType struct{}

func (n NbtPathType) ID() int { return 23 } // minecraft:nbt_path

func (n NbtPathType) Parse(r *commander.CommandReader) (any, error) {
	start := r.Cursor()
	var steps []nbtpath.PathStep

	for r.CanRead() && r.Peek() != ' ' {
		parsed, err := readNbtPathSegment(r)
		if err != nil {
			r.SetCursor(start)
			return nil, err
		}
		steps = append(steps, parsed...)

		if r.CanRead() && r.Peek() == '.' {
			r.Skip()
		}
	}
	if len(steps) == 0 {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}

	return &nbtpath.Path{
		Steps: steps,
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

func readNbtPathSegment(r *commander.CommandReader) ([]nbtpath.PathStep, error) {
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
		return []nbtpath.PathStep{nbtpath.SelfMatch{Filter: filter}}, nil
	}
	if ch == '[' {
		step, err := readNbtPathIndex(r)
		if err != nil {
			return nil, err
		}
		return []nbtpath.PathStep{step}, nil
	}
	name, err := readNbtPathKey(r)
	if err != nil {
		return nil, err
	}
	out := []nbtpath.PathStep{nbtpath.MemberStep{Name: name}}
	selfMatched := false
	for r.CanRead() {
		c := r.Peek()
		if c == '{' && !selfMatched {
			filter, err := readCompoundFilter(r)
			if err != nil {
				return nil, err
			}
			out = append(out, nbtpath.SelfMatch{Filter: filter})
			selfMatched = true
		} else if c == '[' {
			step, err := readNbtPathIndex(r)
			if err != nil {
				return nil, err
			}
			out = append(out, step)
		} else {
			break
		}
	}

	return out, nil
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

func readNbtPathIndex(r *commander.CommandReader) (nbtpath.PathStep, error) {
	r.Skip()
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	ch := r.Peek()
	if ch == ']' {
		r.Skip()
		return nbtpath.AllStep{}, nil
	}
	if ch == '{' {
		filter, err := readCompoundFilter(r)
		if err != nil {
			return nil, err
		}
		if !r.CanRead() || r.Peek() != ']' {
			return nil, commander.NewParsingErrorAt(
				tc.Translatable(mcdata.ArgumentNbtExpectedValue),
				r.Input(), r.Cursor(),
			)
		}
		r.Skip()
		return nbtpath.MatchAll{Filter: filter}, nil
	}
	start := r.Cursor()
	if r.CanRead() && r.Peek() == '-' {
		r.Skip()
	}
	for r.CanRead() && r.Peek() >= '0' && r.Peek() <= '9' {
		r.Skip()
	}
	if r.Cursor() == start {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	idx, err := strconv.Atoi(r.Input()[start:r.Cursor()])
	if err != nil {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), start,
		)
	}
	if !r.CanRead() || r.Peek() != ']' {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedValue),
			r.Input(), r.Cursor(),
		)
	}
	r.Skip()

	return nbtpath.IndexStep{Index: idx}, nil
}

// readCompoundFilter consumes a balanced {...} SNBT compound and returns
// it as a pre-decoded map[string]any. Filter parsing happens once at parse
// time rather than per resolve.
func readCompoundFilter(r *commander.CommandReader) (map[string]any, error) {
	start := r.Cursor()
	if err := readSNBTBalanced(r); err != nil {
		return nil, err
	}
	raw := r.Input()[start:r.Cursor()]
	tagType, err := validateSNBT(raw)
	if err != nil || tagType != nbt.TagCompound {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedCompound),
			r.Input(), start,
		)
	}
	v, err := nbtpath.SNBTToValue(nbt.StringifiedMessage(raw))
	if err != nil {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedCompound),
			r.Input(), start,
		)
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, commander.NewParsingErrorAt(
			tc.Translatable(mcdata.ArgumentNbtExpectedCompound),
			r.Input(), start,
		)
	}
	return m, nil
}
