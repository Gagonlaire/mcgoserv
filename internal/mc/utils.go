package mc

import "math"

const (
	StateHandshake State = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
	StateMax
)

const (
	TicksPerSecond = 20
	TicksPerDay    = 24000
)

const (
	ActionAddPlayer          PlayerAction = 1 << 0 // 1
	ActionInitializeChat     PlayerAction = 1 << 1 // 2
	ActionUpdateGameMode     PlayerAction = 1 << 2 // 4
	ActionUpdateListed       PlayerAction = 1 << 3 // 8
	ActionUpdateLatency      PlayerAction = 1 << 4 // 16
	ActionUpdateDisplayName  PlayerAction = 1 << 5 // 32
	ActionUpdateListPriority PlayerAction = 1 << 6 // 64
	ActionUpdateHat          PlayerAction = 1 << 7 // 128
)

const (
	MaxQuantizedValue = 32766.0
)

var ServerDataPacks = PrefixedArray[DataPackIdentifier]{
	Slice: &[]DataPackIdentifier{
		{
			Namespace: String("minecraft"),
			ID:        String("core"),
			Version:   String(GameVersion),
		},
	},
}

func (v *VarInt) Len() int {
	val := uint32(*v)

	if *v < 0 {
		return 5
	}
	n := 1
	for val >= 0x80 {
		val >>= 7
		n++
	}
	return n
}

func NewPrefixedArray[E any](slice *[]E) *PrefixedArray[E] {
	return &PrefixedArray[E]{
		Slice: slice,
	}
}

func NewPrefixedOptional[E any](value *E) *PrefixedOptional[E] {
	return &PrefixedOptional[E]{
		Has:   value != nil,
		Value: value,
	}
}

func (b *BitSet) Set(i int, value bool) {
	if b.PrefixedArray.Slice == nil {
		return
	}
	data := *b.PrefixedArray.Slice
	idx := i / 64
	off := uint(i % 64)
	if idx >= len(data) {
		return
	}
	if value {
		data[idx] |= 1 << off
	} else {
		data[idx] &^= 1 << off
	}
}

func (b *BitSet) Get(i int) bool {
	if b.PrefixedArray.Slice == nil {
		return false
	}
	data := *b.PrefixedArray.Slice
	idx := i / 64
	off := uint(i % 64)
	if idx >= len(data) {
		return false
	}
	return (data[idx] & (1 << off)) != 0
}

func (P *PrefixedOptional[X]) Set(value *X) {
	P.Value = value
	P.Has = value != nil
}

func NewFixedBitSet(n int) *FixedBitSet {
	byteSize := int(math.Ceil(float64(n) / 8.0))
	return &FixedBitSet{
		BitCount: n,
		Data:     make([]byte, byteSize),
	}
}

func (F *FixedBitSet) Set(i int, value bool) {
	if value {
		F.Data[i/8] |= 1 << (i % 8)
	} else {
		F.Data[i/8] &^= 1 << (i % 8)
	}
}

func (F *FixedBitSet) Get(i int) (bool, error) {
	return (F.Data[i/8] & (1 << (i % 8))) != 0, nil
}

func pack(value float64) int64 {
	return int64(math.Round((value*0.5 + 0.5) * MaxQuantizedValue))
}

func unpack(value uint64) float64 {
	v := float64(value & 32767)
	v = math.Min(v, MaxQuantizedValue)

	return v*2.0/MaxQuantizedValue - 1.0
}
