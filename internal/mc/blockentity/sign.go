package blockentity

import "github.com/Gagonlaire/mcgoserv/internal/world"

// Sign is an inert (non-ticking) block entity. It exists to hold text data, the
// per-tick loop must never see it. TODO: text lines, editing, networking...
type Sign struct {
	pos world.BlockPos
}

func NewSign(pos world.BlockPos) *Sign { return &Sign{pos: pos} }

func (s *Sign) Pos() world.BlockPos { return s.pos }
func (s *Sign) Type() world.BEType  { return TypeSign }

func newSignBE(pos world.BlockPos, _ any) (world.BlockEntity, error) {
	return NewSign(pos), nil
}
