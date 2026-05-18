package nbtpath_test

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

func TestMatch(t *testing.T) {
	t.Run("compound_subset_matches_when_all_keys_present", func(t *testing.T) {
		candidate := map[string]any{"Health": float32(20), "Air": int16(300), "Fire": int16(-1)}
		filter := map[string]any{"Health": float32(20), "Fire": int16(-1)}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("subset filter should match")
		}
	})

	t.Run("compound_missing_key_fails", func(t *testing.T) {
		candidate := map[string]any{"Health": float32(20)}
		filter := map[string]any{"Health": float32(20), "Air": int16(300)}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("missing key should not match")
		}
	})

	t.Run("compound_value_mismatch_fails", func(t *testing.T) {
		candidate := map[string]any{"Health": float32(20)}
		filter := map[string]any{"Health": float32(19)}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("value mismatch should not match")
		}
	})

	t.Run("compound_recursive_match", func(t *testing.T) {
		candidate := map[string]any{
			"Item": map[string]any{"id": "minecraft:slime_ball", "Count": int8(5)},
		}
		filter := map[string]any{
			"Item": map[string]any{"id": "minecraft:slime_ball"},
		}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("nested subset should match")
		}
	})

	t.Run("compound_recursive_mismatch", func(t *testing.T) {
		candidate := map[string]any{
			"Item": map[string]any{"id": "minecraft:slime_ball"},
		}
		filter := map[string]any{
			"Item": map[string]any{"id": "minecraft:stick"},
		}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("inner mismatch should not match")
		}
	})

	t.Run("numeric_tag_strict_byte_vs_int_fails", func(t *testing.T) {
		candidate := map[string]any{"v": int32(0)}
		filter := map[string]any{"v": int8(0)}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("byte filter should not match int candidate (NBT tag strict)")
		}
	})

	t.Run("list_subset_order_independent", func(t *testing.T) {
		candidate := map[string]any{"Tags": []string{"a", "b", "c"}}
		filter := map[string]any{"Tags": []string{"c", "a"}}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("list subset (different order) should match")
		}
	})

	t.Run("list_missing_element_fails", func(t *testing.T) {
		candidate := map[string]any{"Tags": []string{"a", "b"}}
		filter := map[string]any{"Tags": []string{"a", "z"}}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("filter element not in candidate should not match")
		}
	})

	t.Run("list_of_compounds_subset", func(t *testing.T) {
		candidate := map[string]any{
			"Items": []any{
				map[string]any{"id": "minecraft:stick", "Count": int8(1)},
				map[string]any{"id": "minecraft:slime_ball", "Count": int8(64)},
			},
		}
		filter := map[string]any{
			"Items": []any{
				map[string]any{"id": "minecraft:slime_ball"},
			},
		}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("list of compounds subset should match")
		}
	})

	t.Run("empty_filter_compound_matches_any_compound", func(t *testing.T) {
		candidate := map[string]any{"Health": float32(20)}
		filter := map[string]any{}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("empty filter compound should match")
		}
	})

	t.Run("empty_filter_list_matches_any_list", func(t *testing.T) {
		candidate := map[string]any{"Tags": []string{"a", "b"}}
		filter := map[string]any{"Tags": []any{}}
		if !nbtpath.Match(candidate, filter) {
			t.Fatal("empty filter list should match any list")
		}
	})

	t.Run("kind_mismatch_compound_vs_scalar_fails", func(t *testing.T) {
		candidate := map[string]any{"Item": "not a compound"}
		filter := map[string]any{"Item": map[string]any{"id": "x"}}
		if nbtpath.Match(candidate, filter) {
			t.Fatal("compound filter should not match scalar candidate")
		}
	})

	t.Run("scalar_top_level_equals", func(t *testing.T) {
		if !nbtpath.Match(int8(1), int8(1)) {
			t.Fatal("equal scalars should match at top level")
		}
		if nbtpath.Match(int8(1), int8(0)) {
			t.Fatal("unequal scalars should not match")
		}
	})
}
