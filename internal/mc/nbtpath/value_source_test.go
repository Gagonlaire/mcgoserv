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
