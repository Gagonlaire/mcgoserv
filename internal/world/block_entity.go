package world

type BEType uint8

type BlockEntity interface {
	Pos() BlockPos
	Type() BEType
}

type BETickContext struct {
	Dim  *Dimension
	Tick int64
}

type Ticker interface {
	BlockEntity
	Tick(ctx *BETickContext)
}
