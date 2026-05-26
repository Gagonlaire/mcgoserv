package block

import "github.com/Gagonlaire/mcgoserv/internal/world"

// DoorBlock https://minecraft.wiki/w/Door
type DoorBlock struct {
	*DefaultBlock
}

type doorHalf int32

const (
	doorHalfUpper doorHalf = 0
	doorHalfLower doorHalf = 1
)

type doorHinge int32

const (
	doorHingeLeft  doorHinge = 0
	doorHingeRight doorHinge = 1
)

func doorFacingIdx(d world.Direction) int32 {
	switch d {
	case world.DirectionNorth:
		return 0
	case world.DirectionSouth:
		return 1
	case world.DirectionWest:
		return 2
	case world.DirectionEast:
		return 3
	}
	return 0
}

// doorState computes the state ID for a door of `id`, placed closed and
// unpowered. State offset from MinStateID is mixed-radix:
//
//	facing*16 + half*8 + hinge*4 + open*2 + powered*1
//
// Booleans (open, powered) encode as TRUE=0, FALSE=1 in vanilla data.
// TODO: we should make this computation automatic based on 'components'
func doorState(id ID, facing world.Direction, half doorHalf, hinge doorHinge) int32 {
	return int32(id.MinStateID()) +
		doorFacingIdx(facing)*16 +
		int32(half)*8 +
		int32(hinge)*4 +
		3
}

// used for adjacent door detection. Any registered door qualifies, since
// 1.21 (24w21a) doors of different materials pair as double doors instead of
// both placing with the same-side hinge. They do not open together tho.
var doorIDs [len(registry)]bool

func isDoorState(stateID int32) bool {
	id, ok := FromStateID(int(stateID))
	if !ok {
		return false
	}
	return doorIDs[id]
}

// OnBreak removes both halves of the door in one atomic break.
// TODO: neighbor-driven cleanup belongs in OnNeighborChange so that any destructive cause (explosions, pistons) is handled uniformly
// TODO: divergence in current impl and vanilla observations:
//
//	breaking upper part: particles on top
//	breaking lower part: particles on bottom and top
func (b *DoorBlock) OnBreak(ctx world.BreakContext) world.BreakResult {
	otherPos := ctx.Pos.Up()
	if doorHalfFromState(b.ID(), ctx.State) == doorHalfUpper {
		otherPos = ctx.Pos.Down()
	}
	changes := []world.BlockChange{{Pos: ctx.Pos, NewState: 0}}
	if otherState, ok := ctx.View.GetState(otherPos); ok && isDoorState(otherState) {
		changes = append(changes, world.BlockChange{Pos: otherPos, NewState: 0})
	}
	return world.BreakResult{OK: true, Changes: changes}
}

func doorHalfFromState(id ID, state int32) doorHalf {
	offset := state - int32(id.MinStateID())
	return doorHalf((offset / 8) % 2)
}

func (b *DoorBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
	upperPos := ctx.Pos.Up()
	upperState, ok := ctx.View.GetState(upperPos)
	if !ok || !IsReplaceableState(upperState) {
		return world.PlaceResult{OK: false}
	}

	// TODO: vanilla states "Doors must be supported by a full solid block face beneath them.", current impl
	// differ in some edge cases, top slabs should be accepted and stairs/walls/fences should be rejected.
	// Fix when per state collision shape present in block gen data
	belowState, belowOK := ctx.View.GetState(ctx.Pos.Down())
	if !belowOK || !IsSolidFullCube(belowState) {
		return world.PlaceResult{OK: false}
	}

	facing := ctx.PlayerFacing()
	hinge := b.selectHinge(ctx, facing)
	id := b.ID()
	return world.PlaceResult{
		OK: true,
		Writes: []world.BlockChange{
			{Pos: ctx.Pos, NewState: doorState(id, facing, doorHalfLower, hinge)},
			{Pos: upperPos, NewState: doorState(id, facing, doorHalfUpper, hinge)},
		},
	}
}

// selectHinge runs the vanilla hinge-side selection algorithm
func (b *DoorBlock) selectHinge(ctx world.PlaceContext, facing world.Direction) doorHinge {
	ccw := facing.RotateCCW()
	cw := facing.RotateCW()
	leftPos := ctx.Pos.Offset(ccw)
	rightPos := ctx.Pos.Offset(cw)

	leftLower, _ := ctx.View.GetState(leftPos)
	leftUpper, _ := ctx.View.GetState(leftPos.Up())
	rightLower, _ := ctx.View.GetState(rightPos)
	rightUpper, _ := ctx.View.GetState(rightPos.Up())

	if isDoorState(leftLower) || isDoorState(leftUpper) {
		return doorHingeRight
	}
	if isDoorState(rightLower) || isDoorState(rightUpper) {
		return doorHingeLeft
	}

	weight := 0
	if IsSolidFullCube(leftLower) {
		weight++
	}
	if IsSolidFullCube(leftUpper) {
		weight++
	}
	if IsSolidFullCube(rightLower) {
		weight--
	}
	if IsSolidFullCube(rightUpper) {
		weight--
	}
	if weight > 0 {
		return doorHingeLeft
	}
	if weight < 0 {
		return doorHingeRight
	}

	var hitOffset float32
	switch facing {
	case world.DirectionNorth:
		hitOffset = ctx.Hit[0]
	case world.DirectionSouth:
		hitOffset = 1 - ctx.Hit[0]
	case world.DirectionEast:
		hitOffset = ctx.Hit[2]
	case world.DirectionWest:
		hitOffset = 1 - ctx.Hit[2]
	}
	if hitOffset > 0.5 {
		return doorHingeRight
	}
	return doorHingeLeft
}

// TODO: check that all doors has the exact same behavior
var doorNames = [...]string{
	"oak_door",
	"spruce_door",
	"birch_door",
	"jungle_door",
	"acacia_door",
	"cherry_door",
	"dark_oak_door",
	"pale_oak_door",
	"mangrove_door",
	"bamboo_door",
	"crimson_door",
	"warped_door",
	"iron_door",
	"copper_door",
	"exposed_copper_door",
	"weathered_copper_door",
	"oxidized_copper_door",
	"waxed_copper_door",
	"waxed_exposed_copper_door",
	"waxed_weathered_copper_door",
	"waxed_oxidized_copper_door",
}

func registerDoors() {
	for _, name := range doorNames {
		id, ok := FromString(name)
		if !ok {
			continue
		}
		Register(id, &DoorBlock{DefaultBlock: NewDefaultBlock(id)})
		doorIDs[id] = true
	}
}
