package nbtpath_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
)

func TestPathStepKinds(t *testing.T) {
	t.Run("member_step_carries_name", func(t *testing.T) {
		var s nbtpath.PathStep = nbtpath.MemberStep{Name: "Pos"}
		m, ok := s.(nbtpath.MemberStep)
		if !ok {
			t.Fatal("MemberStep does not satisfy PathStep")
		}
		if m.Name != "Pos" {
			t.Fatalf("want %q got %q", "Pos", m.Name)
		}
	})

	t.Run("index_step_carries_index", func(t *testing.T) {
		var s nbtpath.PathStep = nbtpath.IndexStep{Index: 3}
		i, ok := s.(nbtpath.IndexStep)
		if !ok {
			t.Fatal("IndexStep does not satisfy PathStep")
		}
		if i.Index != 3 {
			t.Fatalf("want 3 got %d", i.Index)
		}
	})

	t.Run("all_step_distinct_from_index_zero", func(t *testing.T) {
		steps := []nbtpath.PathStep{nbtpath.AllStep{}, nbtpath.IndexStep{Index: 0}}
		switch steps[0].(type) {
		case nbtpath.AllStep:
		default:
			t.Fatal("first step should be AllStep")
		}
		switch steps[1].(type) {
		case nbtpath.IndexStep:
		default:
			t.Fatal("second step should be IndexStep")
		}
	})

	t.Run("self_match_holds_pre_parsed_filter_map", func(t *testing.T) {
		filter := map[string]any{"id": "creeper"}
		var s nbtpath.PathStep = nbtpath.SelfMatch{Filter: filter}
		sm, ok := s.(nbtpath.SelfMatch)
		if !ok {
			t.Fatal("SelfMatch does not satisfy PathStep")
		}
		if sm.Filter["id"] != "creeper" {
			t.Fatalf("filter not preserved: %#v", sm.Filter)
		}
	})

	t.Run("match_all_holds_pre_parsed_filter_map", func(t *testing.T) {
		var s nbtpath.PathStep = nbtpath.MatchAll{Filter: map[string]any{"Slot": int8(0)}}
		ma, ok := s.(nbtpath.MatchAll)
		if !ok {
			t.Fatal("MatchAll does not satisfy PathStep")
		}
		if ma.Filter["Slot"] != int8(0) {
			t.Fatalf("filter not preserved: %#v", ma.Filter)
		}
	})
}

func TestSentinelErrors(t *testing.T) {
	t.Run("path_not_found_is_distinct_via_errors_is", func(t *testing.T) {
		wrapped := fmt.Errorf("walk failed: %w", nbtpath.ErrPathNotFound)
		if !errors.Is(wrapped, nbtpath.ErrPathNotFound) {
			t.Fatal("errors.Is should match wrapped ErrPathNotFound")
		}
		if errors.Is(wrapped, nbtpath.ErrNotAList) {
			t.Fatal("ErrPathNotFound should not match ErrNotAList")
		}
	})
}
