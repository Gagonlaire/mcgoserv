package world

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
)

type BlockView interface {
	GetState(pos BlockPos) (int32, bool)
}

type BlockChange struct {
	Pos      BlockPos
	OldState int32
	NewState int32
}

// PlaceContext TODO: pool when neighbor/tick paths land.
type PlaceContext struct {
	Pos        BlockPos
	ClickedPos BlockPos
	Face       Direction
	Hit        [3]float32
	Player     *entity.Player
	Hand       entity.Hand
	UsedItem   item.Stack
	View       BlockView
}

type PlaceResult struct {
	OK           bool
	Writes       []BlockChange
	BEAdds       []BlockEntity
	OpenSignEdit *BlockPos
}

func (ctx PlaceContext) PlayerFacing() Direction {
	if ctx.Player == nil {
		return ctx.Face
	}
	return CardinalFromYaw(ctx.Player.Rotation[0])
}

// BreakContext TODO: pool when neighbor/tick paths land.
type BreakContext struct {
	Pos     BlockPos
	State   int32
	Breaker entity.Entity
	Tool    item.Stack
	View    BlockView
}

type BreakResult struct {
	OK         bool
	Changes    []BlockChange
	BERemovals []BlockPos
}

type InteractContext struct {
	Pos      BlockPos
	State    int32
	Player   *entity.Player
	Hand     entity.Hand
	UsedItem item.Stack
	Hit      [3]float32
	Face     Direction
	View     BlockView
}

type InteractResult uint8

const (
	InteractPass InteractResult = iota
	InteractSuccess
	InteractConsume
)

type NeighborContext struct {
	Pos       BlockPos
	From      BlockPos
	FromState int32
	View      BlockView
}

// TickContext TODO: replace rand with seeded rand
type TickContext struct {
	Pos   BlockPos
	State int32
	View  BlockView
}

func (d *Dimension) GetState(pos BlockPos) (int32, bool) {
	state, err := d.GetBlock(pos.X, pos.Y, pos.Z)
	if err != nil {
		return 0, false
	}
	return state, true
}

type BlockBehavior interface {
	OnPlace(ctx PlaceContext) PlaceResult
	OnBreak(ctx BreakContext) BreakResult
}

func (d *Dimension) PlaceBlock(behavior BlockBehavior, ctx *PlaceContext) PlaceResult {
	if ctx.View == nil {
		ctx.View = d
	}
	result := behavior.OnPlace(*ctx)
	if !result.OK {
		return result
	}

	if len(result.BEAdds) > 0 {
		writeSet := make(map[BlockPos]struct{}, len(result.Writes))
		for _, w := range result.Writes {
			writeSet[w.Pos] = struct{}{}
		}
		for _, be := range result.BEAdds {
			if _, ok := writeSet[be.Pos()]; !ok {
				panic(fmt.Sprintf("world: BE spawn at %v is not in PlaceResult.Writes", be.Pos()))
			}
		}
	}
	for i := range result.Writes {
		w := &result.Writes[i]
		w.OldState, _ = d.GetState(w.Pos)
		// TODO: replace this with methods on blocks, so they self clean BE
		d.removeBlockEntity(w.Pos)
		_ = d.SetBlock(w.Pos.X, w.Pos.Y, w.Pos.Z, w.NewState)
	}
	for _, be := range result.BEAdds {
		d.addBlockEntity(be)
	}
	return result
}

func (d *Dimension) BreakBlock(behavior BlockBehavior, ctx *BreakContext) BreakResult {
	if ctx.View == nil {
		ctx.View = d
	}
	result := behavior.OnBreak(*ctx)
	if !result.OK {
		return result
	}
	for i := range result.Changes {
		c := &result.Changes[i]
		c.OldState, _ = d.GetState(c.Pos)
		_ = d.SetBlock(c.Pos.X, c.Pos.Y, c.Pos.Z, c.NewState)
		if c.NewState == 0 {
			d.removeBlockEntity(c.Pos)
		}
	}
	for _, pos := range result.BERemovals {
		d.removeBlockEntity(pos)
	}
	return result
}

func (d *Dimension) removeBlockEntity(pos BlockPos) {
	d.GetChunk(pos.X>>4, pos.Z>>4).RemoveBlockEntity(pos)
}

func (d *Dimension) addBlockEntity(be BlockEntity) {
	pos := be.Pos()
	d.GetChunk(pos.X>>4, pos.Z>>4).SetBlockEntity(be)
}
