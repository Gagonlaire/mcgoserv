package world

import (
	"testing"
)

func TestSectionSingleGetSet(t *testing.T) {
	s := NewSection(0)
	if got := s.Get(0); got != 0 {
		t.Fatalf("fresh Single(0) Get(0) = %d, want 0", got)
	}
	if got := s.BlockCount(); got != 0 {
		t.Fatalf("BlockCount = %d, want 0", got)
	}

	s2 := NewSection(5)
	if got := s2.Get(123); got != 5 {
		t.Fatalf("Single(5) Get(123) = %d, want 5", got)
	}
	if got := s2.BlockCount(); got != SectionSize {
		t.Fatalf("Single(5) BlockCount = %d, want %d", got, SectionSize)
	}
}

func TestSectionSingleToIndirectPromotion(t *testing.T) {
	s := NewSection(0)
	if err := s.Set(42, 7); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := s.impl.(*indirectImpl); !ok {
		t.Fatalf("impl after promotion = %T, want *indirectImpl", s.impl)
	}
	if got := s.Get(42); got != 7 {
		t.Fatalf("Get(42) = %d, want 7", got)
	}
	if got := s.Get(0); got != 0 {
		t.Fatalf("Get(0) after promotion = %d, want 0 (old single value)", got)
	}
	if got := s.BlockCount(); got != 1 {
		t.Fatalf("BlockCount = %d, want 1", got)
	}
}

func TestSectionIndirectToDirectPromotion(t *testing.T) {
	s := NewSection(0)
	// Fill 257 distinct non-air states — forces Indirect to overflow into Direct.
	for i := 0; i < 257; i++ {
		if err := s.Set(i, int32(i+1)); err != nil {
			t.Fatalf("Set(%d): %v", i, err)
		}
	}
	if _, ok := s.impl.(*directImpl); !ok {
		t.Fatalf("impl after 257 unique states = %T, want *directImpl", s.impl)
	}
	for i := 0; i < 257; i++ {
		if got := s.Get(i); got != int32(i+1) {
			t.Fatalf("Get(%d) = %d, want %d", i, got, i+1)
		}
	}
	if got := s.BlockCount(); got != 257 {
		t.Fatalf("BlockCount = %d, want 257", got)
	}
}

func TestSectionFillDemotes(t *testing.T) {
	s := NewSection(0)
	// Push into Indirect.
	_ = s.Set(0, 1)
	_ = s.Set(1, 2)
	if _, ok := s.impl.(*indirectImpl); !ok {
		t.Fatalf("expected Indirect before Fill, got %T", s.impl)
	}
	s.Fill(3)
	if _, ok := s.impl.(*singleImpl); !ok {
		t.Fatalf("impl after Fill = %T, want *singleImpl", s.impl)
	}
	if got := s.BlockCount(); got != SectionSize {
		t.Fatalf("BlockCount after Fill(3) = %d, want %d", got, SectionSize)
	}
	if got := s.Get(2048); got != 3 {
		t.Fatalf("Get(2048) after Fill(3) = %d, want 3", got)
	}
}

func TestSectionAllAirDemotesOnSet(t *testing.T) {
	s := NewSection(0)
	_ = s.Set(0, 9)
	if got := s.BlockCount(); got != 1 {
		t.Fatalf("BlockCount after one set = %d, want 1", got)
	}
	_ = s.Set(0, 0)
	if got := s.BlockCount(); got != 0 {
		t.Fatalf("BlockCount after revert = %d, want 0", got)
	}
	if _, ok := s.impl.(*singleImpl); !ok {
		t.Fatalf("impl after revert = %T, want *singleImpl", s.impl)
	}
}

func TestSectionBlockCountDelta(t *testing.T) {
	s := NewSection(0)
	_ = s.Set(10, 5)
	_ = s.Set(20, 5)
	_ = s.Set(30, 5)
	if got := s.BlockCount(); got != 3 {
		t.Fatalf("BlockCount = %d, want 3", got)
	}
	// Overwrite non-air with non-air — no delta.
	_ = s.Set(10, 7)
	if got := s.BlockCount(); got != 3 {
		t.Fatalf("BlockCount after non-air swap = %d, want 3", got)
	}
	// Remove one.
	_ = s.Set(20, 0)
	if got := s.BlockCount(); got != 2 {
		t.Fatalf("BlockCount after one removal = %d, want 2", got)
	}
}

func TestSectionFillRange(t *testing.T) {
	s := NewSection(0)
	// Fill bottom 4 layers with state 9 → 4 * 256 = 1024 non-air.
	s.FillRange(0, 4, 9)
	if got := s.BlockCount(); got != 4*256 {
		t.Fatalf("BlockCount after FillRange(0,4,9) = %d, want %d", got, 4*256)
	}
	if got := s.Get(0); got != 9 {
		t.Fatalf("Get(0) = %d, want 9", got)
	}
	if got := s.Get(((4) << 8) | 0); got != 0 {
		t.Fatalf("Get(y=4 base) = %d, want 0 (above filled range)", got)
	}
	// Fill the whole range → demote to Single via Fill path.
	s.FillRange(0, 16, 0)
	if _, ok := s.impl.(*singleImpl); !ok {
		t.Fatalf("impl after FillRange(0,16,0) = %T, want *singleImpl", s.impl)
	}
	if got := s.BlockCount(); got != 0 {
		t.Fatalf("BlockCount after FillRange(0,16,0) = %d, want 0", got)
	}
}

func TestSectionIterOrder(t *testing.T) {
	s := NewSection(0)
	// Set a few distinct values.
	_ = s.Set(0, 1)
	_ = s.Set(1, 2)
	_ = s.Set(SectionSize-1, 3)

	seen := 0
	var lastIdx = -1
	s.Iter(func(idx int, state int32) bool {
		if idx != lastIdx+1 {
			t.Fatalf("Iter order broken: got idx %d after %d", idx, lastIdx)
		}
		lastIdx = idx
		switch idx {
		case 0:
			if state != 1 {
				t.Fatalf("Iter idx 0 state = %d, want 1", state)
			}
		case 1:
			if state != 2 {
				t.Fatalf("Iter idx 1 state = %d, want 2", state)
			}
		case SectionSize - 1:
			if state != 3 {
				t.Fatalf("Iter idx last state = %d, want 3", state)
			}
		default:
			if state != 0 {
				t.Fatalf("Iter idx %d state = %d, want 0", idx, state)
			}
		}
		seen++
		return true
	})
	if seen != SectionSize {
		t.Fatalf("Iter visited %d cells, want %d", seen, SectionSize)
	}
}

func TestSectionDirectIter(t *testing.T) {
	s := NewSection(0)
	for i := 0; i < 300; i++ {
		_ = s.Set(i, int32(i+1))
	}
	if _, ok := s.impl.(*directImpl); !ok {
		t.Fatalf("expected Direct, got %T", s.impl)
	}
	count := 0
	s.Iter(func(idx int, state int32) bool {
		if idx < 300 && state != int32(idx+1) {
			t.Fatalf("Direct iter mismatch at %d: got %d, want %d", idx, state, idx+1)
		}
		count++
		return true
	})
	if count != SectionSize {
		t.Fatalf("Direct iter visited %d, want %d", count, SectionSize)
	}
}
