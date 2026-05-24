package world

// LightSection TODO: bench unified vs split LightSection (sky+block locality).
type LightSection struct {
	Sky   [2048]byte
	Block [2048]byte
}

func (l *LightSection) SkyLight(idx int) uint8 {
	b := l.Sky[idx>>1]
	if idx&1 == 0 {
		return b & 0x0F
	}
	return b >> 4
}

func (l *LightSection) SetSkyLight(idx int, v uint8) {
	v &= 0x0F
	bi := idx >> 1
	b := l.Sky[bi]
	if idx&1 == 0 {
		l.Sky[bi] = (b &^ 0x0F) | v
	} else {
		l.Sky[bi] = (b & 0x0F) | (v << 4)
	}
}

func (l *LightSection) BlockLight(idx int) uint8 {
	b := l.Block[idx>>1]
	if idx&1 == 0 {
		return b & 0x0F
	}
	return b >> 4
}

func (l *LightSection) SetBlockLight(idx int, v uint8) {
	v &= 0x0F
	bi := idx >> 1
	b := l.Block[bi]
	if idx&1 == 0 {
		l.Block[bi] = (b &^ 0x0F) | v
	} else {
		l.Block[bi] = (b & 0x0F) | (v << 4)
	}
}
