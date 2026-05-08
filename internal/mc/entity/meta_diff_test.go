package entity

import (
	"testing"
)

func TestMetaSnapshot(t *testing.T) {
	t.Run("captures_all_meta_field_values", func(t *testing.T) {
		b := &BaseEntity{Air: 100, TicksFrozen: 5}
		b.CustomNameVisible = true
		snap := b.MetaSnapshot()
		if snap[IndexAirTicks] != int16(100) {
			t.Fatalf("Air not captured: %v", snap[IndexAirTicks])
		}
		if snap[IndexTicksFrozen] != int32(5) {
			t.Fatalf("TicksFrozen not captured: %v", snap[IndexTicksFrozen])
		}
		if snap[IndexCustomNameVisible] != true {
			t.Fatalf("CustomNameVisible not captured: %v", snap[IndexCustomNameVisible])
		}
	})
}

func TestMetaDiffMark(t *testing.T) {
	t.Run("no_change_does_not_set_dirty", func(t *testing.T) {
		b := &BaseEntity{Air: 300}
		b.MetaDiffMark(b.MetaSnapshot())
		if b.HasMetaChanges() {
			t.Fatalf("no changes expected; bits=%v", b.DirtyTracker)
		}
	})

	t.Run("changing_custom_name_visible_marks_index", func(t *testing.T) {
		b := &BaseEntity{}
		snap := b.MetaSnapshot()
		b.CustomNameVisible = true
		b.MetaDiffMark(snap)
		if !b.IsDirty(IndexCustomNameVisible) {
			t.Fatal("expected IndexCustomNameVisible dirty")
		}
	})

	t.Run("two_changes_mark_two_indices", func(t *testing.T) {
		b := &BaseEntity{Air: 300}
		snap := b.MetaSnapshot()
		b.Air = 100
		b.NoGravity = true
		b.MetaDiffMark(snap)
		if !b.IsDirty(IndexAirTicks) {
			t.Fatal("expected IndexAirTicks dirty")
		}
		if !b.IsDirty(IndexNoGravity) {
			t.Fatal("expected IndexNoGravity dirty")
		}
	})

	t.Run("parent_layer_change_marks_parent_index", func(t *testing.T) {
		l := &LivingEntity{}
		snap := l.MetaSnapshot()
		l.BaseEntity.CustomNameVisible = true
		l.MetaDiffMark(snap)
		if !l.IsDirty(IndexCustomNameVisible) {
			t.Fatal("LivingEntity should propagate BaseEntity diff")
		}
	})
}
