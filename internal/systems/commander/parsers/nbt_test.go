package parsers_test

import (
	"reflect"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func parsePath(t *testing.T, input string) *nbtpath.Path {
	t.Helper()
	r := commander.NewCommandReader(input)
	v, err := parsers.NbtPath.Parse(r)
	if err != nil {
		t.Fatalf("parse %q: %v", input, err)
	}
	p, ok := v.(*nbtpath.Path)
	if !ok {
		t.Fatalf("want *nbtpath.Path got %T", v)
	}
	return p
}

func TestParseNbtPath(t *testing.T) {
	t.Run("single_member_emits_one_member_step", func(t *testing.T) {
		p := parsePath(t, "Health")
		want := []nbtpath.PathStep{nbtpath.MemberStep{Name: "Health"}}
		if !reflect.DeepEqual(p.Steps, want) {
			t.Fatalf("want %#v got %#v", want, p.Steps)
		}
	})

	t.Run("dotted_chain_emits_two_member_steps", func(t *testing.T) {
		p := parsePath(t, "foo.bar")
		want := []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "foo"},
			nbtpath.MemberStep{Name: "bar"},
		}
		if !reflect.DeepEqual(p.Steps, want) {
			t.Fatalf("want %#v got %#v", want, p.Steps)
		}
	})

	t.Run("bracket_index_emits_index_step", func(t *testing.T) {
		p := parsePath(t, "Items[0]")
		want := []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.IndexStep{Index: 0},
		}
		if !reflect.DeepEqual(p.Steps, want) {
			t.Fatalf("want %#v got %#v", want, p.Steps)
		}
	})

	t.Run("empty_brackets_emit_all_step", func(t *testing.T) {
		p := parsePath(t, "Items[]")
		want := []nbtpath.PathStep{
			nbtpath.MemberStep{Name: "Items"},
			nbtpath.AllStep{},
		}
		if !reflect.DeepEqual(p.Steps, want) {
			t.Fatalf("want %#v got %#v", want, p.Steps)
		}
	})

	t.Run("list_filter_emits_match_all_with_decoded_map", func(t *testing.T) {
		p := parsePath(t, "Items[{Slot:0b}]")
		if len(p.Steps) != 2 {
			t.Fatalf("want 2 steps got %d", len(p.Steps))
		}
		if _, ok := p.Steps[0].(nbtpath.MemberStep); !ok {
			t.Fatalf("step 0 not MemberStep: %T", p.Steps[0])
		}
		ma, ok := p.Steps[1].(nbtpath.MatchAll)
		if !ok {
			t.Fatalf("step 1 not MatchAll: %T", p.Steps[1])
		}
		if ma.Filter["Slot"] != int8(0) {
			t.Fatalf("filter not pre-decoded: %#v", ma.Filter)
		}
	})

	t.Run("leading_compound_filter_emits_self_match", func(t *testing.T) {
		p := parsePath(t, `{id:"zombie"}`)
		if len(p.Steps) != 1 {
			t.Fatalf("want 1 step got %d", len(p.Steps))
		}
		sm, ok := p.Steps[0].(nbtpath.SelfMatch)
		if !ok {
			t.Fatalf("step 0 not SelfMatch: %T", p.Steps[0])
		}
		if sm.Filter["id"] != "zombie" {
			t.Fatalf("filter not pre-decoded: %#v", sm.Filter)
		}
	})

	t.Run("key_then_compound_filter_emits_member_then_self_match", func(t *testing.T) {
		p := parsePath(t, `Items{id:"stone"}`)
		if len(p.Steps) != 2 {
			t.Fatalf("want 2 steps got %d", len(p.Steps))
		}
		if m := p.Steps[0].(nbtpath.MemberStep); m.Name != "Items" {
			t.Fatalf("step 0: want Items got %q", m.Name)
		}
		sm, ok := p.Steps[1].(nbtpath.SelfMatch)
		if !ok {
			t.Fatalf("step 1 not SelfMatch: %T", p.Steps[1])
		}
		if sm.Filter["id"] != "stone" {
			t.Fatalf("filter not pre-decoded: %#v", sm.Filter)
		}
	})

	t.Run("quoted_key_with_dot_inside", func(t *testing.T) {
		p := parsePath(t, `"foo.bar"`)
		want := []nbtpath.PathStep{nbtpath.MemberStep{Name: "foo.bar"}}
		if !reflect.DeepEqual(p.Steps, want) {
			t.Fatalf("want %#v got %#v", want, p.Steps)
		}
	})

	t.Run("empty_path_returns_expected_value", func(t *testing.T) {
		r := commander.NewCommandReader("")
		_, err := parsers.NbtPath.Parse(r)
		if err == nil {
			t.Fatal("want error got nil")
		}
	})
}
