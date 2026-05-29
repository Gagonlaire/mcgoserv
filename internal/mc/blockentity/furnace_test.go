package blockentity

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/container"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

const (
	testLitState   int32 = 100
	testUnlitState int32 = 200
)

func newFurnaceFixture(t *testing.T, input, fuel int32) (*Furnace, *world.Dimension, world.BlockPos) {
	t.Helper()
	dim := &world.Dimension{
		Chunks: map[uint64]*world.Chunk{},
		Type:   world.DimensionType{MinY: -64, Height: 384},
	}
	pos := world.BlockPos{X: 0, Y: 64, Z: 0}
	if err := dim.SetBlock(pos.X, pos.Y, pos.Z, testUnlitState); err != nil {
		t.Fatalf("seed unlit block: %v", err)
	}

	f := NewFurnace(pos, testLitState, testUnlitState)
	if input > 0 {
		_ = f.Inventory().Set(furnaceSlotInput, container.Slot{ItemID: int32(item.RawIronID), Count: input})
	}
	if fuel > 0 {
		_ = f.Inventory().Set(furnaceSlotFuel, container.Slot{ItemID: int32(item.CoalID), Count: fuel})
	}
	return f, dim, pos
}

func tickN(f *Furnace, dim *world.Dimension, n int) {
	ctx := &world.BETickContext{Dim: dim, Tick: 0}
	for i := 0; i < n; i++ {
		ctx.Tick = int64(i)
		f.Tick(ctx)
	}
}

func TestFurnace_LightsAndConsumesFuelOnFirstTick(t *testing.T) {
	f, dim, pos := newFurnaceFixture(t, 8, 1)

	tickN(f, dim, 1)

	if !f.isLit() {
		t.Fatal("furnace did not light with valid fuel + input")
	}
	if got := f.Inventory().Get(furnaceSlotFuel).Count; got != 0 {
		t.Fatalf("fuel count = %d after lighting, want 0 (one coal consumed)", got)
	}
	if got, _ := dim.GetBlock(pos.X, pos.Y, pos.Z); got != testLitState {
		t.Fatalf("block state = %d, want lit state %d", got, testLitState)
	}
}

func TestFurnace_SmeltsOneItemAfterCookTime(t *testing.T) {
	f, dim, _ := newFurnaceFixture(t, 8, 1)

	tickN(f, dim, defaultCookTime) // exactly one item's worth of cook ticks

	out := f.Inventory().Get(furnaceSlotOutput)
	if out.ItemID != int32(item.IronIngotID) || out.Count != 1 {
		t.Fatalf("output = {id:%d count:%d}, want one iron ingot (id:%d)", out.ItemID, out.Count, item.IronIngotID)
	}
	if in := f.Inventory().Get(furnaceSlotInput).Count; in != 7 {
		t.Fatalf("input count = %d, want 7 (one consumed)", in)
	}
}

func TestFurnace_DoesNothingWithoutFuel(t *testing.T) {
	f, dim, pos := newFurnaceFixture(t, 8, 0)

	tickN(f, dim, defaultCookTime)

	if f.isLit() {
		t.Fatal("furnace lit without fuel")
	}
	if out := f.Inventory().Get(furnaceSlotOutput).Count; out != 0 {
		t.Fatalf("output count = %d without fuel, want 0", out)
	}
	if got, _ := dim.GetBlock(pos.X, pos.Y, pos.Z); got != testUnlitState {
		t.Fatalf("block state = %d, want unlit %d", got, testUnlitState)
	}
}

func TestFurnace_GoesOutWhenFuelExhausted(t *testing.T) {
	f, dim, pos := newFurnaceFixture(t, 1, 1)

	tickN(f, dim, 1601)

	if f.isLit() {
		t.Fatal("furnace still lit after fuel exhausted")
	}
	if out := f.Inventory().Get(furnaceSlotOutput); out.Count != 1 || out.ItemID != int32(item.IronIngotID) {
		t.Fatalf("output = {id:%d count:%d}, want one iron ingot", out.ItemID, out.Count)
	}
	if got, _ := dim.GetBlock(pos.X, pos.Y, pos.Z); got != testUnlitState {
		t.Fatalf("block state = %d after burnout, want unlit %d", got, testUnlitState)
	}
}

func TestFurnace_ImplementsTicker(t *testing.T) {
	var _ Ticker = (*Furnace)(nil)
	var _ Container = (*Furnace)(nil)
}
