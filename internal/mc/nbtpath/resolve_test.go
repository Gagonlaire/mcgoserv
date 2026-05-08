package nbtpath_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

func TestResolve(t *testing.T) {
	t.Run("empty_path_returns_root_anchor", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		anchors, err := nbtpath.Resolve(root, nbtpath.Path{})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 1 {
			t.Fatalf("want 1 anchor, got %d", len(anchors))
		}
		if !reflect.DeepEqual(anchors[0].Value(), root) {
			t.Fatalf("anchor value should be root, got %#v", anchors[0].Value())
		}
	})

	t.Run("member_step_on_compound_returns_one_anchor", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 1 {
			t.Fatalf("want 1 anchor, got %d", len(anchors))
		}
		a := anchors[0]
		if a.Key != "Health" {
			t.Fatalf("want Key=Health got %q", a.Key)
		}
		if !reflect.DeepEqual(a.Parent, root) {
			t.Fatalf("anchor parent should be root")
		}
		if a.Value() != float32(20) {
			t.Fatalf("anchor value should be 20, got %v", a.Value())
		}
	})
}

func TestResolveErrors(t *testing.T) {
	t.Run("member_missing_key_path_not_found", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Missing"}}}
		_, err := nbtpath.Resolve(root, p)
		if !errors.Is(err, nbtpath.ErrPathNotFound) {
			t.Fatalf("want ErrPathNotFound got %v", err)
		}
	})

	t.Run("member_step_on_list_returns_not_a_compound", func(t *testing.T) {
		root := []any{int32(1), int32(2)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		_, err := nbtpath.Resolve(root, p)
		if !errors.Is(err, nbtpath.ErrNotACompound) {
			t.Fatalf("want ErrNotACompound got %v", err)
		}
	})

	t.Run("index_step_oob_returns_index_out_of_bounds", func(t *testing.T) {
		root := map[string]any{"Items": []any{int32(1), int32(2)}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.IndexStep{Index: 5},
		}}
		_, err := nbtpath.Resolve(root, p)
		if !errors.Is(err, nbtpath.ErrIndexOOB) {
			t.Fatalf("want ErrIndexOOB got %v", err)
		}
	})

	t.Run("index_on_compound_returns_not_a_list", func(t *testing.T) {
		root := map[string]any{"X": int32(1)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.IndexStep{Index: 0}}}
		_, err := nbtpath.Resolve(root, p)
		if !errors.Is(err, nbtpath.ErrNotAList) {
			t.Fatalf("want ErrNotAList got %v", err)
		}
	})
}

func TestResolveIndex(t *testing.T) {
	t.Run("index_step_returns_one_anchor", func(t *testing.T) {
		list := []any{int32(10), int32(20), int32(30)}
		root := map[string]any{"Items": list}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.IndexStep{Index: 1},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 1 {
			t.Fatalf("want 1 anchor, got %d", len(anchors))
		}
		if anchors[0].Index != 1 {
			t.Fatalf("want Index=1 got %d", anchors[0].Index)
		}
		if anchors[0].Value() != int32(20) {
			t.Fatalf("want value 20 got %v", anchors[0].Value())
		}
	})
}

func TestResolveSelfMatch(t *testing.T) {
	t.Run("self_match_filter_subsumes_returns_root", func(t *testing.T) {
		root := map[string]any{"id": "creeper", "Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.SelfMatch{Filter: map[string]any{"id": "creeper"}},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 1 {
			t.Fatalf("want 1 anchor got %d", len(anchors))
		}
	})

	t.Run("self_match_no_match_path_not_found", func(t *testing.T) {
		root := map[string]any{"id": "zombie"}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.SelfMatch{Filter: map[string]any{"id": "creeper"}},
		}}
		_, err := nbtpath.Resolve(root, p)
		if !errors.Is(err, nbtpath.ErrPathNotFound) {
			t.Fatalf("want ErrPathNotFound got %v", err)
		}
	})
}

func TestResolveChained(t *testing.T) {
	t.Run("chained_member_then_index_returns_nested_anchor", func(t *testing.T) {
		root := map[string]any{"Pos": []any{float64(1), float64(2), float64(3)}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Pos"},
			nbtpath.IndexStep{Index: 2},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 1 {
			t.Fatalf("want 1 anchor got %d", len(anchors))
		}
		if anchors[0].Value() != float64(3) {
			t.Fatalf("want 3.0 got %v", anchors[0].Value())
		}
	})

	t.Run("chained_member_then_all_fans_out", func(t *testing.T) {
		root := map[string]any{"Items": []any{
			map[string]any{"id": "a"},
			map[string]any{"id": "b"},
		}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.AllStep{},
			nbtpath.MemberStep{Name: "id"},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 2 {
			t.Fatalf("want 2 anchors got %d", len(anchors))
		}
		if anchors[0].Value() != "a" || anchors[1].Value() != "b" {
			t.Fatalf("want [a,b] got [%v,%v]", anchors[0].Value(), anchors[1].Value())
		}
	})
}

func TestResolveMatchAll(t *testing.T) {
	t.Run("match_all_in_list_returns_only_matching_subset", func(t *testing.T) {
		root := map[string]any{"Items": []any{
			map[string]any{"Slot": int8(0), "id": "stone"},
			map[string]any{"Slot": int8(1), "id": "dirt"},
			map[string]any{"Slot": int8(0), "id": "diamond"},
		}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.MatchAll{Filter: map[string]any{"Slot": int8(0)}},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 2 {
			t.Fatalf("want 2 anchors got %d", len(anchors))
		}
		if anchors[0].Index != 0 || anchors[1].Index != 2 {
			t.Fatalf("want indices [0,2] got [%d,%d]", anchors[0].Index, anchors[1].Index)
		}
	})
}

func TestResolveAll(t *testing.T) {
	t.Run("all_step_three_elements_returns_three_anchors_in_order", func(t *testing.T) {
		root := map[string]any{"Items": []any{int32(10), int32(20), int32(30)}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.AllStep{},
		}}
		anchors, err := nbtpath.Resolve(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(anchors) != 3 {
			t.Fatalf("want 3 anchors, got %d", len(anchors))
		}
		want := []int32{10, 20, 30}
		for i, a := range anchors {
			if a.Index != i {
				t.Fatalf("anchor %d: want Index=%d got %d", i, i, a.Index)
			}
			if a.Value() != want[i] {
				t.Fatalf("anchor %d: want %d got %v", i, want[i], a.Value())
			}
		}
	})
}
