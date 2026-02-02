package mc

const (
	StateHandshake State = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
	StateMax
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
