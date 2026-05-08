package nbtpath

import "reflect"

// Value reads the value at the anchor location. For the root anchor
// (no Key, Index < 0) it returns Parent itself.
func (a Anchor) Value() any {
	if a.Key != "" {
		if m, ok := a.Parent.(map[string]any); ok {
			return m[a.Key]
		}
		return nil
	}
	if a.Index >= 0 {
		if s, ok := toAnySlice(a.Parent); ok {
			return s[a.Index]
		}
		return nil
	}
	return a.Parent
}

func compoundMatch(candidate, filter map[string]any) bool {
	for k, fv := range filter {
		cv, exists := candidate[k]
		if !exists {
			return false
		}
		if fMap, ok := fv.(map[string]any); ok {
			cMap, ok2 := cv.(map[string]any)
			if !ok2 || !compoundMatch(cMap, fMap) {
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

// Resolve walks the path against root and returns one anchor per match.
// An empty path yields a single root anchor.
func Resolve(root any, p Path) ([]Anchor, error) {
	current := []Anchor{{Parent: root, Index: -1}}
	for _, step := range p.Steps {
		next, err := stepResolve(current, step)
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func stepResolve(in []Anchor, step PathStep) ([]Anchor, error) {
	var out []Anchor
	for _, a := range in {
		v := a.Value()
		switch s := step.(type) {
		case MemberStep:
			m, ok := v.(map[string]any)
			if !ok {
				return nil, ErrNotACompound
			}
			if _, exists := m[s.Name]; !exists {
				return nil, ErrPathNotFound
			}
			out = append(out, Anchor{Parent: m, Key: s.Name, Index: -1})
		case IndexStep:
			list, ok := toAnySlice(v)
			if !ok {
				return nil, ErrNotAList
			}
			if s.Index < 0 || s.Index >= len(list) {
				return nil, ErrIndexOOB
			}
			out = append(out, Anchor{Parent: v, Index: s.Index})
		case AllStep:
			list, ok := toAnySlice(v)
			if !ok {
				return nil, ErrNotAList
			}
			for i := range list {
				out = append(out, Anchor{Parent: v, Index: i})
			}
		case SelfMatch:
			cm, ok := v.(map[string]any)
			if !ok {
				return nil, ErrNotACompound
			}
			if !compoundMatch(cm, s.Filter) {
				return nil, ErrPathNotFound
			}
			out = append(out, a)
		case MatchAll:
			list, ok := toAnySlice(v)
			if !ok {
				return nil, ErrNotAList
			}
			for i, elem := range list {
				cm, ok := elem.(map[string]any)
				if !ok {
					continue
				}
				if compoundMatch(cm, s.Filter) {
					out = append(out, Anchor{Parent: v, Index: i})
				}
			}
		}
	}
	return out, nil
}
