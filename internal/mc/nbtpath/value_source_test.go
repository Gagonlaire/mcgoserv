package nbtpath_test

import (
	"errors"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

type fakeSource struct {
	root any
}

func (f fakeSource) NbtRoot() (any, error) { return f.root, nil }
func (f fakeSource) NbtGet(p nbtpath.Path) (any, error) {
	anchors, err := nbtpath.Resolve(f.root, p)
	if err != nil {
		return nil, err
	}
	if len(anchors) == 0 {
		return nil, nbtpath.ErrPathNotFound
	}
	return anchors[0].Value(), nil
}

func TestLiteralValueSource(t *testing.T) {
	t.Run("returns_single_value", func(t *testing.T) {
		src := nbtpath.LiteralValueSource{Value: int32(42)}
		out, err := src.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(out) != 1 || out[0] != int32(42) {
			t.Fatalf("want [42] got %#v", out)
		}
	})
}

func TestFromValueSource(t *testing.T) {
	t.Run("single_anchor_returns_one_value", func(t *testing.T) {
		src := fakeSource{root: map[string]any{"Health": float32(20)}}
		fv := nbtpath.FromValueSource{
			Src:  src,
			Path: nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}},
		}
		out, err := fv.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(out) != 1 || out[0] != float32(20) {
			t.Fatalf("want [20] got %#v", out)
		}
	})

	t.Run("multi_anchor_returns_list", func(t *testing.T) {
		src := fakeSource{root: map[string]any{"Items": []any{
			map[string]any{"id": "a"},
			map[string]any{"id": "b"},
		}}}
		fv := nbtpath.FromValueSource{
			Src: src,
			Path: nbtpath.Path{Steps: []nbtpath.PathStep{
				nbtpath.MemberStep{Name: "Items"},
				nbtpath.AllStep{},
				nbtpath.MemberStep{Name: "id"},
			}},
		}
		out, err := fv.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(out) != 2 || out[0] != "a" || out[1] != "b" {
			t.Fatalf("want [a,b] got %#v", out)
		}
	})

	t.Run("missing_path_returns_path_not_found", func(t *testing.T) {
		src := fakeSource{root: map[string]any{"Health": float32(20)}}
		fv := nbtpath.FromValueSource{
			Src:  src,
			Path: nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Missing"}}},
		}
		_, err := fv.Resolve()
		if !errors.Is(err, nbtpath.ErrPathNotFound) {
			t.Fatalf("want ErrPathNotFound got %v", err)
		}
	})
}

func TestStringValueSource(t *testing.T) {
	mkSrc := func(root any) fakeSource { return fakeSource{root: root} }
	mkPath := func(name string) nbtpath.Path {
		return nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: name}}}
	}
	ip := func(v int) *int { return &v }

	t.Run("whole_string_when_no_bounds", func(t *testing.T) {
		sv := nbtpath.StringValueSource{Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Name")}
		out, err := sv.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(out) != 1 || out[0] != "Alice" {
			t.Fatalf("want [Alice] got %#v", out)
		}
	})

	t.Run("substring_with_start_and_end", func(t *testing.T) {
		sv := nbtpath.StringValueSource{
			Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Name"),
			Start: ip(1), End: ip(4),
		}
		out, err := sv.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out[0] != "lic" {
			t.Fatalf("want lic got %q", out[0])
		}
	})

	t.Run("substring_with_only_start", func(t *testing.T) {
		sv := nbtpath.StringValueSource{
			Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Name"),
			Start: ip(2),
		}
		out, err := sv.Resolve()
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if out[0] != "ice" {
			t.Fatalf("want ice got %q", out[0])
		}
	})

	t.Run("path_not_found", func(t *testing.T) {
		sv := nbtpath.StringValueSource{Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Missing")}
		_, err := sv.Resolve()
		if !errors.Is(err, nbtpath.ErrPathNotFound) {
			t.Fatalf("want ErrPathNotFound got %v", err)
		}
	})

	t.Run("non_string_value", func(t *testing.T) {
		sv := nbtpath.StringValueSource{Src: mkSrc(map[string]any{"Health": float32(20)}), Path: mkPath("Health")}
		_, err := sv.Resolve()
		if !errors.Is(err, nbtpath.ErrNotAString) {
			t.Fatalf("want ErrNotAString got %v", err)
		}
	})

	t.Run("out_of_bounds_rejected", func(t *testing.T) {
		sv := nbtpath.StringValueSource{
			Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Name"),
			Start: ip(0), End: ip(99),
		}
		_, err := sv.Resolve()
		if !errors.Is(err, nbtpath.ErrStringBounds) {
			t.Fatalf("want ErrStringBounds got %v", err)
		}
	})

	t.Run("inverted_bounds_rejected", func(t *testing.T) {
		sv := nbtpath.StringValueSource{
			Src: mkSrc(map[string]any{"Name": "Alice"}), Path: mkPath("Name"),
			Start: ip(3), End: ip(1),
		}
		_, err := sv.Resolve()
		if !errors.Is(err, nbtpath.ErrStringBounds) {
			t.Fatalf("want ErrStringBounds got %v", err)
		}
	})

	t.Run("multiple_values_rejected", func(t *testing.T) {
		sv := nbtpath.StringValueSource{
			Src: mkSrc(map[string]any{"Items": []any{
				map[string]any{"id": "a"},
				map[string]any{"id": "b"},
			}}),
			Path: nbtpath.Path{Steps: []nbtpath.PathStep{
				nbtpath.MemberStep{Name: "Items"},
				nbtpath.AllStep{},
				nbtpath.MemberStep{Name: "id"},
			}},
		}
		_, err := sv.Resolve()
		if !errors.Is(err, nbtpath.ErrMultipleValues) {
			t.Fatalf("want ErrMultipleValues got %v", err)
		}
	})
}
