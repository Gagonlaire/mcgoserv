package world

import "testing"

type fakeBE struct{ pos BlockPos }

func (f *fakeBE) Pos() BlockPos { return f.pos }
func (f *fakeBE) Type() BEType  { return 0 }

type fakeTicker struct {
	fakeBE
	ticks   int
	lastCtx *BETickContext
}

func (f *fakeTicker) Tick(ctx *BETickContext) {
	f.ticks++
	f.lastCtx = ctx
}

func newTestChunk() *Chunk { return NewChunk(0, 0, 24, -64) }

func TestSetBlockEntity_NonTickerStaysOutOfTickerSlice(t *testing.T) {
	c := newTestChunk()
	pos := BlockPos{X: 1, Y: 64, Z: 2}
	be := &fakeBE{pos: pos}

	c.SetBlockEntity(be)

	if got := c.BlockEntity(pos); got != be {
		t.Fatalf("BlockEntity(%v) = %v, want stored BE", pos, got)
	}
	if len(c.tickers) != 0 {
		t.Fatalf("non-ticking BE leaked into tickers: len = %d, want 0", len(c.tickers))
	}
}

func TestSetBlockEntity_TickerJoinsBothMapAndSlice(t *testing.T) {
	c := newTestChunk()
	pos := BlockPos{X: 3, Y: 70, Z: 4}
	tk := &fakeTicker{fakeBE: fakeBE{pos: pos}}

	c.SetBlockEntity(tk)

	if got := c.BlockEntity(pos); got != tk {
		t.Fatalf("BlockEntity(%v) = %v, want stored ticker", pos, got)
	}
	if len(c.tickers) != 1 {
		t.Fatalf("ticker not in slice: len = %d, want 1", len(c.tickers))
	}
}

func TestSetBlockEntity_ReplaceDoesNotDuplicateTicker(t *testing.T) {
	c := newTestChunk()
	pos := BlockPos{X: 0, Y: 0, Z: 0}
	c.SetBlockEntity(&fakeTicker{fakeBE: fakeBE{pos: pos}})
	c.SetBlockEntity(&fakeTicker{fakeBE: fakeBE{pos: pos}})

	if len(c.tickers) != 1 {
		t.Fatalf("replacing a ticker duplicated it: len = %d, want 1", len(c.tickers))
	}
}

func TestRemoveBlockEntity_DropsFromMapAndSlice(t *testing.T) {
	c := newTestChunk()
	pos := BlockPos{X: 5, Y: 64, Z: 6}
	c.SetBlockEntity(&fakeTicker{fakeBE: fakeBE{pos: pos}})

	c.RemoveBlockEntity(pos)

	if got := c.BlockEntity(pos); got != nil {
		t.Fatalf("BlockEntity(%v) = %v after remove, want nil", pos, got)
	}
	if len(c.tickers) != 0 {
		t.Fatalf("ticker not removed: len = %d, want 0", len(c.tickers))
	}
}

func TestChunkTick_DrivesOnlyTickers(t *testing.T) {
	c := newTestChunk()
	tk := &fakeTicker{fakeBE: fakeBE{pos: BlockPos{X: 1, Y: 64, Z: 1}}}
	c.SetBlockEntity(tk)
	c.SetBlockEntity(&fakeBE{pos: BlockPos{X: 2, Y: 64, Z: 2}})

	c.Tick(&BETickContext{Tick: 1})

	if tk.ticks != 1 {
		t.Fatalf("ticker ticked %d times, want 1", tk.ticks)
	}
}

func TestBlockEntities_IteratesAll(t *testing.T) {
	c := newTestChunk()
	c.SetBlockEntity(&fakeTicker{fakeBE: fakeBE{pos: BlockPos{X: 1, Y: 64, Z: 1}}})
	c.SetBlockEntity(&fakeBE{pos: BlockPos{X: 2, Y: 64, Z: 2}})

	count := 0
	for pos, be := range c.BlockEntities() {
		if be.Pos() != pos {
			t.Fatalf("iter key %v != be.Pos() %v", pos, be.Pos())
		}
		count++
	}
	if count != 2 {
		t.Fatalf("iterated %d BEs, want 2", count)
	}
}

func TestDimensionTick_TicksChunksWithServerTick(t *testing.T) {
	d := &Dimension{
		Chunks: map[uint64]*Chunk{},
		Type:   DimensionType{MinY: -64, Height: 384},
	}
	c := d.GetChunk(0, 0)
	tk := &fakeTicker{fakeBE: fakeBE{pos: BlockPos{X: 0, Y: 64, Z: 0}}}
	c.SetBlockEntity(tk)

	d.Tick(7)

	if tk.ticks != 1 {
		t.Fatalf("ticker ticked %d times, want 1", tk.ticks)
	}
	if tk.lastCtx == nil || tk.lastCtx.Tick != 7 {
		t.Fatalf("ctx.Tick = %v, want 7", tk.lastCtx)
	}
	if tk.lastCtx.Dim != d {
		t.Fatalf("ctx.Dim = %p, want %p", tk.lastCtx.Dim, d)
	}
}
