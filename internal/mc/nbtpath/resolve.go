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

func Match(candidate, filter any) bool {
	if fMap, ok := filter.(map[string]any); ok {
		cMap, ok := candidate.(map[string]any)
		if !ok {
			return false
		}
		for k, fv := range fMap {
			cv, exists := cMap[k]
			if !exists {
				return false
			}
			if !Match(cv, fv) {
				return false
			}
		}
		return true
	}
	if fList, ok := toAnySlice(filter); ok {
		cList, ok := toAnySlice(candidate)
		if !ok {
			return false
		}
		for _, fv := range fList {
			matched := false
			for _, cv := range cList {
				if Match(cv, fv) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true
	}
	return reflect.DeepEqual(candidate, filter)
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
			if !Match(cm, s.Filter) {
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
				if Match(cm, s.Filter) {
					out = append(out, Anchor{Parent: v, Index: i})
				}
			}
		}
	}
	return out, nil
}
