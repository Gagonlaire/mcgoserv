package entity

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

func TestZombieNbtRoot(t *testing.T) {
	t.Run("dump_includes_pos_and_health", func(t *testing.T) {
		z := NewZombie("overworld", [3]float64{1, 64, 2}, [2]float32{0, 0})
		root, err := z.NbtRoot()
		if err != nil {
			t.Fatalf("NbtRoot: %v", err)
		}
		m, ok := root.(map[string]any)
		if !ok {
			t.Fatalf("want map got %T", root)
		}
		if _, ok := m["Pos"]; !ok {
			t.Fatalf("expected Pos key, got %v", keys(m))
		}
		if _, ok := m["Health"]; !ok {
			t.Fatalf("expected Health key, got %v", keys(m))
		}
	})
}

func TestZombieNbtGet(t *testing.T) {
	t.Run("path_navigates_to_field", func(t *testing.T) {
		z := NewZombie("overworld", [3]float64{1, 64, 2}, [2]float32{0, 0})
		path := nbtpath.Path{Steps: []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Pos"},
			nbtpath.IndexStep{Index: 0},
		}}
		v, err := z.NbtGet(path)
		if err != nil {
			t.Fatalf("NbtGet: %v", err)
		}
		if v != float64(1) {
			t.Fatalf("want Pos[0]=1.0 got %v", v)
		}
	})
}

func TestZombieNbtMerge(t *testing.T) {
	// Regression for the silent-bug: NBT merge that touches a metadata-tracked
	// field MUST mark the dirty tracker so the metadata sync packet flushes.
	t.Run("merge_custom_name_visible_marks_dirty_tracker", func(t *testing.T) {
		z := NewZombie("overworld", [3]float64{0, 64, 0}, [2]float32{0, 0})
		z.ClearMetaChanges()

		_, err := z.NbtMerge(map[string]any{"CustomNameVisible": true})
		if err != nil {
			t.Fatalf("NbtMerge: %v", err)
		}
		if !z.CustomNameVisible {
			t.Fatal("CustomNameVisible should be true after merge")
		}
		if !z.IsDirty(IndexCustomNameVisible) {
			t.Fatal("IndexCustomNameVisible should be dirty after merge")
		}
	})
}

func TestZombieNbtSet(t *testing.T) {
	t.Run("literal_value_writes_field", func(t *testing.T) {
		z := NewZombie("overworld", [3]float64{0, 64, 0}, [2]float32{0, 0})
		z.ClearMetaChanges()

		path := nbtpath.Path{Steps: []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}}
		n, err := z.NbtSet(path, float32(12.5))
		if err != nil {
			t.Fatalf("NbtSet: %v", err)
		}
		if n != 1 {
			t.Fatalf("want affected=1 got %d", n)
		}
		if z.Health != float32(12.5) {
			t.Fatalf("Health not written: %v", z.Health)
		}
		if !z.IsDirty(IndexHealth) {
			t.Fatal("IndexHealth should be dirty after NbtSet")
		}
	})
}

func TestPlayerNbtSource(t *testing.T) {
	t.Run("player_satisfies_nbt_source", func(t *testing.T) {
		var v any = (*Player)(nil)
		if _, ok := v.(nbtpath.NbtSource); !ok {
			t.Fatal("Player must satisfy NbtSource")
		}
	})
}

func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
