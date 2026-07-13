package block

import (
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mc/blockentity"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

// StandingSignBlock https://minecraft.wiki/w/Sign, the free-standing variant. The
// sign item resolves to this block; OnPlace branches to the wall variant's state
// when the clicked face is horizontal (mirroring vanilla SignItem).
type StandingSignBlock struct {
	*DefaultBlock
	wallID ID
}

// WallSignBlock is the wall-mounted variant. It is never placed from an item directly
// (it has no item form) StandingSignBlock.OnPlace writes its state, so it only needs
// default break/interact behavior. Its BE is the same kind as the standing sign.
type WallSignBlock struct {
	*DefaultBlock
}

// standingSignState: offset from MinStateID is mixed-radix rotation*2 + waterlogged,
// where waterlogged is a TRUE=0/FALSE=1 boolean. We always place un-waterlogged (+1).
func standingSignState(id ID, rotation int32) int32 {
	return int32(id.MinStateID()) + rotation*2 + 1
}

// wallSignState: offset is facingIdx*2 + waterlogged (facing order N/S/W/E matches
// doorFacingIdx). Always un-waterlogged (+1).
func wallSignState(id ID, facing world.Direction) int32 {
	return int32(id.MinStateID()) + doorFacingIdx(facing)*2 + 1
}

// rotation16FromYaw maps a yaw to the 0-15 sign rotation (vanilla RotationSegment).
func rotation16FromYaw(yaw float32) int32 {
	return int32(math.Floor(float64(yaw)*16.0/360.0+0.5)) & 15
}

func (b *StandingSignBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
	if ctx.Face == world.DirectionDown {
		return world.PlaceResult{OK: false}
	}

	pos := ctx.Pos
	var state int32
	if ctx.Face == world.DirectionUp {
		below, ok := ctx.View.GetState(pos.Down())
		if !ok || !IsSolidFullCube(below) {
			return world.PlaceResult{OK: false}
		}
		state = standingSignState(b.ID(), rotation16FromYaw(playerYaw(ctx)+180))
	} else {
		facing := ctx.Face
		behind, ok := ctx.View.GetState(pos.Offset(facing.Opposite()))
		if !ok || !IsSolidFullCube(behind) {
			return world.PlaceResult{OK: false}
		}
		state = wallSignState(b.wallID, facing)
	}

	sign := blockentity.NewSign(pos)
	if ctx.Player != nil {
		sign.SetEditor(ctx.Player.EntityID)
	}
	bePos := pos
	return world.PlaceResult{
		OK:           true,
		Writes:       []world.BlockChange{{Pos: pos, NewState: state}},
		BEAdds:       []world.BlockEntity{sign},
		OpenSignEdit: &bePos,
	}
}

func playerYaw(ctx world.PlaceContext) float32 {
	if ctx.Player == nil {
		return 0
	}
	return ctx.Player.Rotation[0]
}

// signWoods covers every wood with standing+wall sign blocks in the generated data.
// All variants share identical behavior. Hanging signs are a distinct BE kind and
// are deferred (future work).
var signWoods = [...]string{
	"oak", "spruce", "birch", "jungle", "acacia", "cherry",
	"dark_oak", "mangrove", "pale_oak", "bamboo", "crimson", "warped",
}

func registerSigns() {
	for _, wood := range signWoods {
		standingID, ok := FromString(wood + "_sign")
		if !ok {
			continue
		}
		wallID, ok := FromString(wood + "_wall_sign")
		if !ok {
			continue
		}
		Register(standingID, &StandingSignBlock{DefaultBlock: NewDefaultBlock(standingID), wallID: wallID})
		Register(wallID, &WallSignBlock{DefaultBlock: NewDefaultBlock(wallID)})
	}
}
