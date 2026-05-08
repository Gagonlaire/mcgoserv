package nbtpath_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

func TestSet(t *testing.T) {
	t.Run("single_anchor_sets_field_returns_one", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		n, err := nbtpath.Set(root, p, float32(5))
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		if root["Health"] != float32(5) {
			t.Fatalf("want 5 got %v", root["Health"])
		}
	})

	t.Run("missing_intermediate_returns_path_not_found", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Missing"},
			nbtpath.MemberStep{Name: "Sub"},
		}}
		_, err := nbtpath.Set(root, p, int32(1))
		if !errors.Is(err, nbtpath.ErrPathNotFound) {
			t.Fatalf("want ErrPathNotFound got %v", err)
		}
	})

	t.Run("match_all_writes_all_returns_three", func(t *testing.T) {
		root := map[string]any{"Items": []any{
			map[string]any{"Slot": int8(0), "id": "stone"},
			map[string]any{"Slot": int8(0), "id": "dirt"},
			map[string]any{"Slot": int8(0), "id": "wood"},
		}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.AllStep{},
			nbtpath.MemberStep{Name: "id"},
		}}
		n, err := nbtpath.Set(root, p, "diamond")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 3 {
			t.Fatalf("want 3 got %d", n)
		}
		items := root["Items"].([]any)
		for i, it := range items {
			m := it.(map[string]any)
			if m["id"] != "diamond" {
				t.Fatalf("item %d: want diamond got %v", i, m["id"])
			}
		}
	})

	t.Run("index_step_writes_to_indexed_position", func(t *testing.T) {
		root := map[string]any{"Pos": []any{float64(1), float64(2), float64(3)}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Pos"},
			nbtpath.IndexStep{Index: 1},
		}}
		n, err := nbtpath.Set(root, p, float64(99))
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		pos := root["Pos"].([]any)
		if pos[1] != float64(99) {
			t.Fatalf("want 99 got %v", pos[1])
		}
	})
}

func TestAppend(t *testing.T) {
	t.Run("list_member_grows_returns_one", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Tags"}}}
		n, err := nbtpath.Append(root, p, "c")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		got := root["Tags"].([]any)
		want := []any{"a", "b", "c"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	})

	t.Run("non_list_target_returns_expected_list", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		_, err := nbtpath.Append(root, p, float32(1))
		if !errors.Is(err, nbtpath.ErrNotAList) {
			t.Fatalf("want ErrNotAList got %v", err)
		}
	})
}

func TestPrepend(t *testing.T) {
	t.Run("element_appears_at_index_zero", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Tags"}}}
		n, err := nbtpath.Prepend(root, p, "z")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		got := root["Tags"].([]any)
		want := []any{"z", "a", "b"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("want %v got %v", want, got)
		}
	})
}

func TestInsert(t *testing.T) {
	t.Run("at_index_zero_shifts_existing", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Tags"}}}
		n, err := nbtpath.Insert(root, p, 0, "z")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		want := []any{"z", "a", "b"}
		if !reflect.DeepEqual(root["Tags"], want) {
			t.Fatalf("want %v got %v", want, root["Tags"])
		}
	})

	t.Run("at_len_acts_like_append", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Tags"}}}
		_, err := nbtpath.Insert(root, p, 2, "c")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		want := []any{"a", "b", "c"}
		if !reflect.DeepEqual(root["Tags"], want) {
			t.Fatalf("want %v got %v", want, root["Tags"])
		}
	})

	t.Run("index_oob_returns_invalid_index", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Tags"}}}
		_, err := nbtpath.Insert(root, p, 99, "x")
		if !errors.Is(err, nbtpath.ErrIndexOOB) {
			t.Fatalf("want ErrIndexOOB got %v", err)
		}
	})
}

func TestRemove(t *testing.T) {
	t.Run("member_deletes_key_from_compound", func(t *testing.T) {
		root := map[string]any{"Health": float32(20), "Air": int16(300)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Air"}}}
		n, err := nbtpath.Remove(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		if _, exists := root["Air"]; exists {
			t.Fatalf("Air should be deleted, root=%v", root)
		}
		if root["Health"] != float32(20) {
			t.Fatal("Health was lost")
		}
	})

	t.Run("index_removes_from_list", func(t *testing.T) {
		root := map[string]any{"Tags": []any{"a", "b", "c"}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Tags"},
			nbtpath.IndexStep{Index: 1},
		}}
		n, err := nbtpath.Remove(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		want := []any{"a", "c"}
		if !reflect.DeepEqual(root["Tags"], want) {
			t.Fatalf("want %v got %v", want, root["Tags"])
		}
	})

	t.Run("match_all_removes_all_returns_count", func(t *testing.T) {
		root := map[string]any{"Items": []any{
			map[string]any{"Slot": int8(0), "id": "stone"},
			map[string]any{"Slot": int8(1), "id": "dirt"},
			map[string]any{"Slot": int8(0), "id": "wood"},
		}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.MatchAll{Filter: map[string]any{"Slot": int8(0)}},
		}}
		n, err := nbtpath.Remove(root, p)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 2 {
			t.Fatalf("want 2 got %d", n)
		}
		items := root["Items"].([]any)
		if len(items) != 1 {
			t.Fatalf("want 1 item left got %d", len(items))
		}
		if items[0].(map[string]any)["id"] != "dirt" {
			t.Fatalf("wrong remaining item: %v", items[0])
		}
	})
}

func TestMerge(t *testing.T) {
	t.Run("merge_root_preserves_untouched_keys", func(t *testing.T) {
		root := map[string]any{"Health": float32(20), "Air": int16(300)}
		_, err := nbtpath.MergeRoot(root, map[string]any{"Health": float32(10)})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if root["Health"] != float32(10) {
			t.Fatalf("Health not updated: %v", root["Health"])
		}
		if root["Air"] != int16(300) {
			t.Fatalf("Air should be untouched: %v", root["Air"])
		}
	})

	t.Run("merge_at_non_compound_target_returns_expected_compound", func(t *testing.T) {
		root := map[string]any{"Health": float32(20)}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		_, err := nbtpath.MergeAt(root, p, map[string]any{"a": int32(1)})
		if !errors.Is(err, nbtpath.ErrNotACompound) {
			t.Fatalf("want ErrNotACompound got %v", err)
		}
	})

	t.Run("merge_at_compound_combines_keys", func(t *testing.T) {
		root := map[string]any{"Inv": map[string]any{"a": int32(1), "b": int32(2)}}
		p := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Inv"}}}
		n, err := nbtpath.MergeAt(root, p, map[string]any{"b": int32(20), "c": int32(3)})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if n != 1 {
			t.Fatalf("want 1 got %d", n)
		}
		inv := root["Inv"].(map[string]any)
		want := map[string]any{"a": int32(1), "b": int32(20), "c": int32(3)}
		if !reflect.DeepEqual(inv, want) {
			t.Fatalf("want %v got %v", want, inv)
		}
	})
}
