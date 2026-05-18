package nbtpath

// setAnchor writes v at the anchor's location. Returns ErrSelectsRoot for
// the root anchor (caller should use a root-level merge instead).
func setAnchor(a Anchor, v any) error {
	if a.Key != "" {
		m, ok := a.Parent.(map[string]any)
		if !ok {
			return ErrPathNotFound
		}
		m[a.Key] = v
		return nil
	}
	if a.Index >= 0 {
		list, ok := a.Parent.([]any)
		if !ok {
			return ErrNotAList
		}
		if a.Index >= len(list) {
			return ErrIndexOOB
		}
		list[a.Index] = v
		return nil
	}
	return ErrSelectsRoot
}

// Set overwrites the value at every anchor matched by p with v.
func Set(root any, p Path, v any) (int, error) {
	anchors, err := Resolve(root, p)
	if err != nil {
		return 0, err
	}
	for _, a := range anchors {
		if err := setAnchor(a, v); err != nil {
			return 0, err
		}
	}
	return len(anchors), nil
}

// Append appends v to every list anchor matched by p.
func Append(root any, p Path, v any) (int, error) {
	anchors, err := Resolve(root, p)
	if err != nil {
		return 0, err
	}
	for _, a := range anchors {
		list, ok := toAnySlice(a.Value())
		if !ok {
			return 0, ErrNotAList
		}
		if err := setAnchor(a, append(list, v)); err != nil {
			return 0, err
		}
	}
	return len(anchors), nil
}

// Prepend prepends v to every list anchor matched by p.
func Prepend(root any, p Path, v any) (int, error) {
	anchors, err := Resolve(root, p)
	if err != nil {
		return 0, err
	}
	for _, a := range anchors {
		list, ok := toAnySlice(a.Value())
		if !ok {
			return 0, ErrNotAList
		}
		newList := make([]any, 0, len(list)+1)
		newList = append(newList, v)
		newList = append(newList, list...)
		if err := setAnchor(a, newList); err != nil {
			return 0, err
		}
	}
	return len(anchors), nil
}

// Remove deletes every tag matched by p. For map-key paths it deletes the
// key; for index/match paths on lists it rebuilds the list without the
// matched elements.
func Remove(root any, p Path) (int, error) {
	if len(p.Steps) == 0 {
		return 0, ErrSelectsRoot
	}
	last := p.Steps[len(p.Steps)-1]
	parentPath := Path{Steps: p.Steps[:len(p.Steps)-1]}
	parents, err := Resolve(root, parentPath)
	if err != nil {
		return 0, err
	}

	affected := 0
	for _, pa := range parents {
		v := pa.Value()
		switch s := last.(type) {
		case MemberStep:
			m, ok := v.(map[string]any)
			if !ok {
				return 0, ErrNotACompound
			}
			if _, exists := m[s.Name]; exists {
				delete(m, s.Name)
				affected++
			}
		case IndexStep:
			list, ok := v.([]any)
			if !ok {
				return 0, ErrNotAList
			}
			if s.Index < 0 || s.Index >= len(list) {
				return 0, ErrIndexOOB
			}
			newList := make([]any, 0, len(list)-1)
			newList = append(newList, list[:s.Index]...)
			newList = append(newList, list[s.Index+1:]...)
			if err := setAnchor(pa, newList); err != nil {
				return 0, err
			}
			affected++
		case AllStep:
			list, ok := v.([]any)
			if !ok {
				return 0, ErrNotAList
			}
			affected += len(list)
			if err := setAnchor(pa, []any{}); err != nil {
				return 0, err
			}
		case MatchAll:
			list, ok := v.([]any)
			if !ok {
				return 0, ErrNotAList
			}
			kept := make([]any, 0, len(list))
			removed := 0
			for _, elem := range list {
				if cm, ok := elem.(map[string]any); ok && compoundMatch(cm, s.Filter) {
					removed++
					continue
				}
				kept = append(kept, elem)
			}
			if err := setAnchor(pa, kept); err != nil {
				return 0, err
			}
			affected += removed
		case SelfMatch:
			return 0, ErrSelectsRoot
		}
	}
	return affected, nil
}

// MergeRoot recursively merges src into root at the top level.
func MergeRoot(root map[string]any, src map[string]any) (int, error) {
	mergeCompound(root, src)
	return 1, nil
}

// MergeAt merges src into every compound anchor matched by p.
func MergeAt(root any, p Path, src map[string]any) (int, error) {
	anchors, err := Resolve(root, p)
	if err != nil {
		return 0, err
	}
	for _, a := range anchors {
		dst, ok := a.Value().(map[string]any)
		if !ok {
			return 0, ErrNotACompound
		}
		mergeCompound(dst, src)
	}
	return len(anchors), nil
}

func mergeCompound(dst, src map[string]any) {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				mergeCompound(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
}

// Insert inserts v at idx in every list anchor matched by p.
func Insert(root any, p Path, idx int, v any) (int, error) {
	anchors, err := Resolve(root, p)
	if err != nil {
		return 0, err
	}
	for _, a := range anchors {
		list, ok := toAnySlice(a.Value())
		if !ok {
			return 0, ErrNotAList
		}
		if idx < 0 || idx > len(list) {
			return 0, ErrIndexOOB
		}
		newList := make([]any, 0, len(list)+1)
		newList = append(newList, list[:idx]...)
		newList = append(newList, v)
		newList = append(newList, list[idx:]...)
		if err := setAnchor(a, newList); err != nil {
			return 0, err
		}
	}
	return len(anchors), nil
}
