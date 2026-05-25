package block

import "github.com/Gagonlaire/mcgoserv/internal/world"

type Behavior interface {
	ID() ID
	OnPlace(ctx world.PlaceContext) world.PlaceResult
	OnBreak(ctx world.BreakContext) world.BreakResult
	OnInteract(ctx world.InteractContext) world.InteractResult
	OnNeighborChange(ctx world.NeighborContext) world.PlaceResult
	OnRandomTick(ctx world.TickContext) world.PlaceResult
	OnScheduledTick(ctx world.TickContext) world.PlaceResult
}

type DefaultBlock struct {
	id ID
}

func NewDefaultBlock(id ID) *DefaultBlock {
	return &DefaultBlock{id: id}
}

func (d *DefaultBlock) ID() ID { return d.id }

func (d *DefaultBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
	return world.PlaceResult{
		OK: true,
		Writes: []world.BlockChange{{
			Pos:      ctx.Pos,
			NewState: int32(d.id.DefaultStateID()),
		}},
	}
}

func (d *DefaultBlock) OnBreak(ctx world.BreakContext) world.BreakResult {
	return world.BreakResult{
		OK: true,
		Changes: []world.BlockChange{{
			Pos:      ctx.Pos,
			NewState: 0,
		}},
	}
}

func (d *DefaultBlock) OnInteract(world.InteractContext) world.InteractResult {
	return world.InteractPass
}

func (d *DefaultBlock) OnNeighborChange(world.NeighborContext) world.PlaceResult {
	return world.PlaceResult{}
}

func (d *DefaultBlock) OnRandomTick(world.TickContext) world.PlaceResult {
	return world.PlaceResult{}
}

func (d *DefaultBlock) OnScheduledTick(world.TickContext) world.PlaceResult {
	return world.PlaceResult{}
}
