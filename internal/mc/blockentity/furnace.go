package blockentity

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/container"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

// Furnace slot layout (vanilla container component order).
const (
	furnaceSlotInput  = 0
	furnaceSlotFuel   = 1
	furnaceSlotOutput = 2
)

// defaultCookTime is the ticks an item takes to smelt (10s at 20 TPS).
const defaultCookTime = 200

// smeltRecipes is a hardcoded subset of vanilla smelting recipes (input item ->
// result item). TODO: load from mcdata recipe JSON once the recipe system lands.
var smeltRecipes = map[int32]int32{
	int32(item.RawIronID):     int32(item.IronIngotID),
	int32(item.RawCopperID):   int32(item.CopperIngotID),
	int32(item.RawGoldID):     int32(item.GoldIngotID),
	int32(item.SandID):        int32(item.GlassID),
	int32(item.CobblestoneID): int32(item.StoneID),
}

// fuelBurnTicks is a hardcoded subset of vanilla fuels (item -> ticks of burn
// time per unit). TODO: load from item burn-time data once available.
var fuelBurnTicks = map[int32]int{
	int32(item.CoalID):       1600,
	int32(item.CharcoalID):   1600,
	int32(item.CoalBlockID):  16000,
	int32(item.LavaBucketID): 20000,
	int32(item.StickID):      100,
	int32(item.OakPlanksID):  300,
}

// Furnace is a ticking, container block entity that smelts an input item using a
// fuel item, mirroring vanilla furnace mechanics closely enough to exercise the
// full tick pipeline. litState/unlitState are the block state IDs to toggle when
// the furnace lights or goes out; the owning block computes them at placement so
// this package never needs to import the block package.
// TODO: should also store xp
type Furnace struct {
	pos              world.BlockPos
	inv              container.Instance
	litState         int32
	unlitState       int32
	litTimeRemaining int // ticks left on the currently burning fuel
	litTotalTime     int // full burn time of that fuel (for progress display)
	cookingTimeSpent int // ticks the current item has been smelting
	cookingTotalTime int // ticks required to finish (defaultCookTime)
}

func NewFurnace(pos world.BlockPos, litState, unlitState int32) *Furnace {
	return &Furnace{
		pos:              pos,
		inv:              container.NewInstance(3),
		litState:         litState,
		unlitState:       unlitState,
		cookingTotalTime: defaultCookTime,
	}
}

func (f *Furnace) Pos() world.BlockPos            { return f.pos }
func (f *Furnace) Type() world.BEType             { return TypeFurnace }
func (f *Furnace) Inventory() *container.Instance { return &f.inv }

func (f *Furnace) isLit() bool { return f.litTimeRemaining > 0 }

// Tick advances one smelting step. It follows the vanilla order: burn down the
// current fuel, (re)light if there is fuel and a smeltable input, advance the
// cook timer and produce a result on completion, then push any lit/unlit block
// state change to the world.
func (f *Furnace) Tick(ctx *world.BETickContext) {
	wasLit := f.isLit()
	if f.litTimeRemaining > 0 {
		f.litTimeRemaining--
	}

	input := f.inv.Get(furnaceSlotInput)
	result, canSmelt := f.smeltResult(input)

	if !f.isLit() && canSmelt {
		f.consumeFuel()
	}

	if f.isLit() && canSmelt {
		f.cookingTimeSpent++
		if f.cookingTimeSpent >= f.cookingTotalTime {
			f.cookingTimeSpent = 0
			f.smeltOne(result)
		}
	} else if f.cookingTimeSpent > 0 {
		// Progress decays when not actively smelting (vanilla loses 2/tick).
		f.cookingTimeSpent -= 2
		if f.cookingTimeSpent < 0 {
			f.cookingTimeSpent = 0
		}
	}

	if nowLit := f.isLit(); nowLit != wasLit {
		state := f.unlitState
		if nowLit {
			state = f.litState
		}
		_ = ctx.Dim.SetBlock(f.pos.X, f.pos.Y, f.pos.Z, state)
	}
}

func (f *Furnace) smeltResult(input container.Slot) (int32, bool) {
	if input.Count <= 0 {
		return 0, false
	}
	result, ok := smeltRecipes[input.ItemID]
	if !ok {
		return 0, false
	}
	out := f.inv.Get(furnaceSlotOutput)
	if out.Count <= 0 {
		return result, true
	}
	if out.ItemID != result {
		return 0, false
	}
	if int(out.Count) >= item.ID(out.ItemID).StackSize() {
		return 0, false
	}
	return result, true
}

func (f *Furnace) consumeFuel() {
	fuel := f.inv.Get(furnaceSlotFuel)
	burn, ok := fuelBurnTicks[fuel.ItemID]
	if !ok || fuel.Count <= 0 {
		return
	}
	f.litTimeRemaining = burn
	f.litTotalTime = burn
	fuel.Count--
	if fuel.Count <= 0 {
		fuel = container.Slot{}
	}
	_ = f.inv.Set(furnaceSlotFuel, fuel)
}

func (f *Furnace) smeltOne(result int32) {
	input := f.inv.Get(furnaceSlotInput)
	input.Count--
	if input.Count <= 0 {
		input = container.Slot{}
	}
	_ = f.inv.Set(furnaceSlotInput, input)

	out := f.inv.Get(furnaceSlotOutput)
	if out.Count <= 0 {
		out = container.Slot{ItemID: result, Count: 1}
	} else {
		out.Count++
	}
	_ = f.inv.Set(furnaceSlotOutput, out)
}
