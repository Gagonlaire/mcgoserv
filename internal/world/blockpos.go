package world

type BlockPos struct {
	X, Y, Z int
}

func (p BlockPos) Offset(d Direction) BlockPos {
	dx, dy, dz := d.Vector()
	return BlockPos{X: p.X + dx, Y: p.Y + dy, Z: p.Z + dz}
}

func (p BlockPos) Up() BlockPos { return BlockPos{X: p.X, Y: p.Y + 1, Z: p.Z} }

func (p BlockPos) Down() BlockPos { return BlockPos{X: p.X, Y: p.Y - 1, Z: p.Z} }
