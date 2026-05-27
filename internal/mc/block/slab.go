package block

import "github.com/Gagonlaire/mcgoserv/internal/world"

// SlabBlock https://minecraft.wiki/w/Slab
type SlabBlock struct {
	*DefaultBlock
}

type slabType int32

const (
	slabTop    slabType = 0
	slabBottom slabType = 1
	slabDouble slabType = 2
)

// slabState computes the state ID for a slab of `id` with the given type.
// State offset from MinStateID is mixed-radix: type*2 + waterlogged.
// `waterlogged` is encoded as TRUE=0, FALSE=1 (empty Values []string{}, project
// convention). We hardcode FALSE here.
//
// TODO: real waterlog support needs fluid detection at placement and flow
// handling in the empty half. Bundle with the fluid subsystem.
// TODO: replace with generic state-computation derived from the block's
// `States []StateProperty` data.
func slabState(id ID, t slabType) int32 {
	return int32(id.MinStateID()) + int32(t)*2 + 1
}

func slabTypeFromState(id ID, state int32) slabType {
	offset := state - int32(id.MinStateID())
	return slabType(offset / 2)
}

func slabHalfAt(face world.Direction, hitY float32, inClickedCell bool) slabType {
	switch face {
	case world.DirectionUp:
		if inClickedCell {
			return slabTop
		}
		return slabBottom
	case world.DirectionDown:
		if inClickedCell {
			return slabBottom
		}
		return slabTop
	default:
		if hitY > 0.5 {
			return slabTop
		}
		return slabBottom
	}
}

var slabIDs [len(registry)]bool

func slabIDFromState(stateID int32) (ID, bool) {
	id, ok := FromStateID(int(stateID))
	if !ok {
		return 0, false
	}
	if !slabIDs[id] {
		return 0, false
	}
	return id, true
}

// OnPlace handles two cases:
//   - merge-in-place: the clicked cell holds a matching slab whose empty half
//     aligns with the click face; the cell becomes a double slab.
//   - normal placement: writes a top or bottom slab in the face-offset cell
//     if that cell is replaceable.
//
// TODO: vanilla also shifts placement into a replaceable clicked cell (e.g.
// click grass with a slab → replace grass with bottom slab). That belongs in
// the server place flow (replaceClicked semantics), not here. Tracked
// alongside the future CanBeReplaced hook.
func (b *SlabBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
	id := b.ID()
	if b.canMergeAtClicked(ctx) {
		return world.PlaceResult{
			OK: true,
			Writes: []world.BlockChange{
				{Pos: ctx.ClickedPos, NewState: slabState(id, slabDouble)},
			},
		}
	}
	targetState, ok := ctx.View.GetState(ctx.Pos)
	if !ok || !IsReplaceableState(targetState) {
		return world.PlaceResult{OK: false}
	}
	half := slabHalfAt(ctx.Face, ctx.Hit[1], false)
	return world.PlaceResult{
		OK: true,
		Writes: []world.BlockChange{
			{Pos: ctx.Pos, NewState: slabState(id, half)},
		},
	}
}

func (b *SlabBlock) canMergeAtClicked(ctx world.PlaceContext) bool {
	state, ok := ctx.View.GetState(ctx.ClickedPos)
	if !ok {
		return false
	}
	cellID, isSlab := slabIDFromState(state)
	if !isSlab || cellID != b.ID() {
		return false
	}
	existing := slabTypeFromState(cellID, state)
	if existing == slabDouble {
		return false
	}
	return slabHalfAt(ctx.Face, ctx.Hit[1], true) != existing
}

// TODO: verify every slab shares identical placement behavior (waterlog
// timing across wood/stone/copper, etc.), same TODO as doors.
// TODO: obscure vanilla case skipped, placing a slab onto a lower slab when
// a matching slab of the same type sits one or two half-cells above causes
// the upper slab to become a double. Cross-cell, cross-type lookup; revisit
// after the common cases are stable.
var slabNames = [...]string{
	"oak_slab", "spruce_slab", "birch_slab", "jungle_slab",
	"acacia_slab", "cherry_slab", "dark_oak_slab", "pale_oak_slab",
	"mangrove_slab", "bamboo_slab", "bamboo_mosaic_slab",
	"crimson_slab", "warped_slab",
	"petrified_oak_slab",
	"stone_slab", "smooth_stone_slab",
	"sandstone_slab", "cut_sandstone_slab", "smooth_sandstone_slab",
	"red_sandstone_slab", "cut_red_sandstone_slab", "smooth_red_sandstone_slab",
	"cobblestone_slab", "mossy_cobblestone_slab",
	"stone_brick_slab", "mossy_stone_brick_slab",
	"brick_slab", "mud_brick_slab", "nether_brick_slab", "red_nether_brick_slab",
	"quartz_slab", "smooth_quartz_slab",
	"granite_slab", "polished_granite_slab",
	"diorite_slab", "polished_diorite_slab",
	"andesite_slab", "polished_andesite_slab",
	"end_stone_brick_slab", "purpur_slab",
	"prismarine_slab", "prismarine_brick_slab", "dark_prismarine_slab",
	"blackstone_slab", "polished_blackstone_slab", "polished_blackstone_brick_slab",
	"cobbled_deepslate_slab", "polished_deepslate_slab",
	"deepslate_brick_slab", "deepslate_tile_slab",
	"tuff_slab", "polished_tuff_slab", "tuff_brick_slab",
	"resin_brick_slab",
	"cut_copper_slab", "exposed_cut_copper_slab",
	"weathered_cut_copper_slab", "oxidized_cut_copper_slab",
	"waxed_cut_copper_slab", "waxed_exposed_cut_copper_slab",
	"waxed_weathered_cut_copper_slab", "waxed_oxidized_cut_copper_slab",
}

func registerSlabs() {
	for _, name := range slabNames {
		id, ok := FromString(name)
		if !ok {
			continue
		}
		Register(id, &SlabBlock{DefaultBlock: NewDefaultBlock(id)})
		slabIDs[id] = true
	}
}
