package block

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

type pendingWrite struct {
	pos   world.BlockPos
	state int32
}

type pendingBE struct {
	pos    world.BlockPos
	beType int32
}

// PlaceContext transaction struct passed to Behavior.OnPlace
type PlaceContext struct {
	world         *world.World
	dim           *world.Dimension
	pos           world.BlockPos
	face          world.Direction
	hit           [3]float32
	player        *entity.Player
	hand          entity.Hand
	usedItem      item.Stack
	writes        []pendingWrite
	blockEntities []pendingBE
}

func NewPlaceContext(
	w *world.World,
	dim *world.Dimension,
	pos world.BlockPos,
	face world.Direction,
	hit [3]float32,
	player *entity.Player,
	hand entity.Hand,
	usedItem item.Stack,
) *PlaceContext {
	return &PlaceContext{
		world:    w,
		dim:      dim,
		pos:      pos,
		face:     face,
		hit:      hit,
		player:   player,
		hand:     hand,
		usedItem: usedItem,
	}
}

func (c *PlaceContext) World() *world.World { return c.world }

func (c *PlaceContext) Pos() world.BlockPos { return c.pos }

func (c *PlaceContext) Face() world.Direction { return c.face }

func (c *PlaceContext) Hit() [3]float32 { return c.hit }

func (c *PlaceContext) Player() *entity.Player { return c.player }

func (c *PlaceContext) Hand() entity.Hand { return c.hand }

func (c *PlaceContext) UsedItem() item.Stack { return c.usedItem }

func (c *PlaceContext) PlayerFacing() world.Direction {
	if c.player == nil {
		return world.DirectionNorth
	}
	return world.CardinalFromYaw(c.player.Rotation[0])
}

func (c *PlaceContext) Set(pos world.BlockPos, state int32) {
	c.writes = append(c.writes, pendingWrite{pos: pos, state: state})
}

// CreateBlockEntity TODO: implement BE
func (c *PlaceContext) CreateBlockEntity(pos world.BlockPos, beType int32) {
	c.blockEntities = append(c.blockEntities, pendingBE{pos: pos, beType: beType})
}

// Commit buffered writes, if error on commit, applied writes are not rollback
func (c *PlaceContext) Commit() error {
	for _, w := range c.writes {
		if err := c.dim.SetBlock(w.pos.X, w.pos.Y, w.pos.Z, w.state); err != nil {
			return err
		}
	}
	for _, be := range c.blockEntities {
		chunk := c.dim.GetChunk(be.pos.X>>4, be.pos.Z>>4)
		chunk.BlockEntities = append(chunk.BlockEntities, world.BlockEntity{
			X:    int8(be.pos.X & 15),
			Y:    int16(be.pos.Y),
			Z:    int8(be.pos.Z & 15),
			Type: be.beType,
		})
	}
	c.writes = c.writes[:0]
	c.blockEntities = c.blockEntities[:0]
	return nil
}

func (c *PlaceContext) Rollback() {
	c.writes = c.writes[:0]
	c.blockEntities = c.blockEntities[:0]
}

// BreakContext same as PlaceContext
type BreakContext struct {
	world   *world.World
	dim     *world.Dimension
	pos     world.BlockPos
	state   int32
	breaker entity.Entity
	tool    item.Stack
}

func NewBreakContext(
	w *world.World,
	dim *world.Dimension,
	pos world.BlockPos,
	state int32,
	breaker entity.Entity,
	tool item.Stack,
) *BreakContext {
	return &BreakContext{world: w, dim: dim, pos: pos, state: state, breaker: breaker, tool: tool}
}

func (c *BreakContext) World() *world.World { return c.world }

func (c *BreakContext) Dim() *world.Dimension { return c.dim }

func (c *BreakContext) Pos() world.BlockPos { return c.pos }

func (c *BreakContext) State() int32 { return c.state }

func (c *BreakContext) Breaker() entity.Entity { return c.breaker }

func (c *BreakContext) Tool() item.Stack { return c.tool }

// InteractContext same as place/brake context
// TODO: check if mob interaction should use this path, if so: player -> entity
type InteractContext struct {
	world  *world.World
	dim    *world.Dimension
	pos    world.BlockPos
	state  int32
	player *entity.Player
	hand   entity.Hand
	item   item.Stack
	hit    [3]float32
	face   world.Direction
}

func NewInteractContext(
	w *world.World,
	dim *world.Dimension,
	pos world.BlockPos,
	state int32,
	player *entity.Player,
	hand entity.Hand,
	usedItem item.Stack,
	hit [3]float32,
	face world.Direction,
) *InteractContext {
	return &InteractContext{
		world:  w,
		dim:    dim,
		pos:    pos,
		state:  state,
		player: player,
		hand:   hand,
		item:   usedItem,
		hit:    hit,
		face:   face,
	}
}

func (c *InteractContext) World() *world.World { return c.world }

func (c *InteractContext) Dim() *world.Dimension { return c.dim }

func (c *InteractContext) Pos() world.BlockPos { return c.pos }

func (c *InteractContext) State() int32 { return c.state }

func (c *InteractContext) Player() *entity.Player { return c.player }

func (c *InteractContext) Hand() entity.Hand { return c.hand }

func (c *InteractContext) UsedItem() item.Stack { return c.item }

func (c *InteractContext) Hit() [3]float32 { return c.hit }

func (c *InteractContext) Face() world.Direction { return c.face }

type NeighborContext struct {
	world     *world.World
	dim       *world.Dimension
	pos       world.BlockPos
	from      world.BlockPos
	fromState int32
}

func NewNeighborContext(
	w *world.World,
	dim *world.Dimension,
	pos, from world.BlockPos,
	fromState int32,
) *NeighborContext {
	return &NeighborContext{world: w, dim: dim, pos: pos, from: from, fromState: fromState}
}

func (c *NeighborContext) World() *world.World   { return c.world }
func (c *NeighborContext) Dim() *world.Dimension { return c.dim }
func (c *NeighborContext) Pos() world.BlockPos   { return c.pos }
func (c *NeighborContext) From() world.BlockPos  { return c.from }
func (c *NeighborContext) FromState() int32      { return c.fromState }

// TickContext TODO: replace rand with seed based rand
type TickContext struct {
	world *world.World
	dim   *world.Dimension
	pos   world.BlockPos
	state int32
}

func NewTickContext(
	w *world.World,
	dim *world.Dimension,
	pos world.BlockPos,
	state int32,
) *TickContext {
	return &TickContext{world: w, dim: dim, pos: pos, state: state}
}

func (c *TickContext) World() *world.World { return c.world }

func (c *TickContext) Dim() *world.Dimension { return c.dim }

func (c *TickContext) Pos() world.BlockPos { return c.pos }

func (c *TickContext) State() int32 { return c.state }
