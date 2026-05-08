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
