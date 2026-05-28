package world

type BEType uint8

type BlockEntity interface {
	Pos() BlockPos
	Type() BEType
}
