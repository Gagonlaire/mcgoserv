package block

type Behavior interface {
	ID() ID
	OnPlace(ctx *PlaceContext) bool
	OnBreak(ctx *BreakContext)
	OnInteract(ctx *InteractContext) InteractResult
	OnNeighborChange(ctx *NeighborContext)
	OnRandomTick(ctx *TickContext)
	OnScheduledTick(ctx *TickContext)
}

type InteractResult uint8

const (
	InteractPass InteractResult = iota
	InteractSuccess
	InteractConsume
)

type DefaultBlock struct {
	id ID
}

func NewDefaultBlock(id ID) *DefaultBlock {
	return &DefaultBlock{id: id}
}

func (d *DefaultBlock) ID() ID { return d.id }

func (d *DefaultBlock) OnPlace(ctx *PlaceContext) bool {
	ctx.Set(ctx.Pos(), int32(d.id.DefaultStateID()))
	return true
}

func (d *DefaultBlock) OnBreak(*BreakContext) {}

func (d *DefaultBlock) OnInteract(*InteractContext) InteractResult {
	return InteractPass
}

func (d *DefaultBlock) OnNeighborChange(*NeighborContext) {}

func (d *DefaultBlock) OnRandomTick(*TickContext) {}

func (d *DefaultBlock) OnScheduledTick(*TickContext) {}
