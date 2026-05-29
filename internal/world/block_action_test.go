package world

import "testing"

type fakeBehavior struct{ place PlaceResult }

func (f *fakeBehavior) OnPlace(PlaceContext) PlaceResult { return f.place }

func (f *fakeBehavior) OnBreak(BreakContext) BreakResult { return BreakResult{} }

func newTestDimension() *Dimension {
	return &Dimension{
		Chunks: map[uint64]*Chunk{},
		Type:   DimensionType{MinY: -64, Height: 384},
	}
}

func (d *Dimension) blockEntityAt(pos BlockPos) BlockEntity {
	return d.GetChunk(pos.X>>4, pos.Z>>4).BlockEntity(pos)
}

func TestPlaceBlock_SpawnInWrites_Attaches(t *testing.T) {
	d := newTestDimension()
	pos := BlockPos{X: 1, Y: 100, Z: 2}
	be := &fakeBE{pos: pos}
	behavior := &fakeBehavior{place: PlaceResult{
		OK:     true,
		Writes: []BlockChange{{Pos: pos, NewState: 5}},
		BEAdds: []BlockEntity{be},
	}}

	d.PlaceBlock(behavior, &PlaceContext{Pos: pos})

	if got := d.blockEntityAt(pos); got != be {
		t.Fatalf("BlockEntity(%v) = %v, want attached spawn", pos, got)
	}
	if state, _ := d.GetState(pos); state != 5 {
		t.Fatalf("state at %v = %d, want 5", pos, state)
	}
}

func TestPlaceBlock_NotOK_NoAttach_NoMutation(t *testing.T) {
	d := newTestDimension()
	pos := BlockPos{X: 3, Y: 100, Z: 4}
	behavior := &fakeBehavior{place: PlaceResult{
		OK:     false,
		Writes: []BlockChange{{Pos: pos, NewState: 5}},
		BEAdds: []BlockEntity{&fakeBE{pos: pos}},
	}}

	d.PlaceBlock(behavior, &PlaceContext{Pos: pos})

	if got := d.blockEntityAt(pos); got != nil {
		t.Fatalf("BlockEntity(%v) = %v on OK=false, want nil", pos, got)
	}
	if state, _ := d.GetState(pos); state != 0 {
		t.Fatalf("state at %v = %d on OK=false, want 0 (no mutation)", pos, state)
	}
}

func TestPlaceBlock_Overwrite_StripsExistingBE(t *testing.T) {
	d := newTestDimension()
	pos := BlockPos{X: 5, Y: 100, Z: 6}
	d.addBlockEntity(&fakeBE{pos: pos})

	// A different block overwrites the BE's position, with no replacement spawn.
	behavior := &fakeBehavior{place: PlaceResult{
		OK:     true,
		Writes: []BlockChange{{Pos: pos, NewState: 9}},
	}}

	d.PlaceBlock(behavior, &PlaceContext{Pos: pos})

	if got := d.blockEntityAt(pos); got != nil {
		t.Fatalf("BlockEntity(%v) = %v after overwrite, want nil (stripped)", pos, got)
	}
	if state, _ := d.GetState(pos); state != 9 {
		t.Fatalf("state at %v = %d, want 9", pos, state)
	}
}

func TestPlaceBlock_SpawnOutsideWrites_Panics(t *testing.T) {
	d := newTestDimension()
	writePos := BlockPos{X: 1, Y: 100, Z: 1}
	spawnPos := BlockPos{X: 2, Y: 100, Z: 2}
	behavior := &fakeBehavior{place: PlaceResult{
		OK:     true,
		Writes: []BlockChange{{Pos: writePos, NewState: 5}},
		BEAdds: []BlockEntity{&fakeBE{pos: spawnPos}},
	}}

	defer func() {
		if recover() == nil {
			t.Fatal("PlaceBlock did not panic on spawn outside Writes")
		}
		// Validation runs before any mutation: the write must not have applied.
		if state, _ := d.GetState(writePos); state != 0 {
			t.Fatalf("state at %v = %d after panic, want 0 (no mutation)", writePos, state)
		}
	}()

	d.PlaceBlock(behavior, &PlaceContext{Pos: writePos})
}
