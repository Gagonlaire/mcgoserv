package nbtpath

// ValueSource resolves to a slice of values
type ValueSource interface {
	Resolve() ([]any, error)
}

// LiteralValueSource carries a single literal value parsed from `value <nbt>`.
type LiteralValueSource struct {
	Value any
}

func (l LiteralValueSource) Resolve() ([]any, error) {
	return []any{l.Value}, nil
}

// FromValueSource reads from another NbtSource at Path.
type FromValueSource struct {
	Src  NbtSource
	Path Path
}

func (f FromValueSource) Resolve() ([]any, error) {
	root, err := f.Src.NbtRoot()
	if err != nil {
		return nil, err
	}
	anchors, err := Resolve(root, f.Path)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(anchors))
	for i, a := range anchors {
		out[i] = a.Value()
	}
	return out, nil
}

type StringValueSource struct {
	Src   NbtSource
	Path  Path
	Start *int
	End   *int
}

func (s StringValueSource) Resolve() ([]any, error) {
	root, err := s.Src.NbtRoot()
	if err != nil {
		return nil, err
	}
	anchors, err := Resolve(root, s.Path)
	if err != nil {
		return nil, err
	}
	if len(anchors) == 0 {
		return nil, ErrPathNotFound
	}
	if len(anchors) > 1 {
		return nil, ErrMultipleValues
	}
	str, ok := anchors[0].Value().(string)
	if !ok {
		return nil, ErrNotAString
	}
	start, end := 0, len(str)
	if s.Start != nil {
		start = *s.Start
	}
	if s.End != nil {
		end = *s.End
	}
	if start < 0 || end > len(str) || start > end {
		return nil, ErrStringBounds
	}
	return []any{str[start:end]}, nil
}
