package block

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/blockentity"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

// FurnaceBlock https://minecraft.wiki/w/Furnace
type FurnaceBlock struct {
	*DefaultBlock
}

// furnaceState computes the state ID for a furnace. Offset from MinStateID is
// mixed-radix: facing*2 + lit, where lit encodes TRUE=0, FALSE=1 (vanilla order,
// confirmed against the generated block report).
func furnaceState(id ID, facing world.Direction, lit bool) int32 {
	off := doorFacingIdx(facing) * 2
	if !lit {
		off++
	}
	return int32(id.MinStateID()) + off
}

func (b *FurnaceBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
	id := b.ID()
	facing := ctx.PlayerFacing().Opposite()
	unlit := furnaceState(id, facing, false)
	lit := furnaceState(id, facing, true)
	return world.PlaceResult{
		OK:     true,
		Writes: []world.BlockChange{{Pos: ctx.Pos, NewState: unlit}},
		BEAdds: []world.BlockEntity{blockentity.NewFurnace(ctx.Pos, lit, unlit)},
	}
}

func newFurnaceBE(pos world.BlockPos, _ any) (world.BlockEntity, error) {
	id, _ := FromString("furnace")
	lit := furnaceState(id, world.DirectionNorth, true)
	unlit := furnaceState(id, world.DirectionNorth, false)
	return blockentity.NewFurnace(pos, lit, unlit), nil
}

func registerFurnace() {
	id, ok := FromString("furnace")
	if !ok {
		return
	}
	Register(id, &FurnaceBlock{DefaultBlock: NewDefaultBlock(id)})
	blockentity.Register(blockentity.TypeFurnace, newFurnaceBE)
}
