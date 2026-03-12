package mc

import (
	"iter"
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

const (
	StateHandshake State = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
	StateMax
)

const (
	StatusStartDigging   = 0
	StatusCancelDigging  = 1
	StatusFinishDigging  = 2
	StatusDropItemStack  = 3
	StatusDropItem       = 4
	StatusReleaseUseItem = 5
	StatusSwapHand       = 6
)

const (
	PoseStanding Pose = iota
	PoseFallFlying
	PoseSleeping
	PoseSwimming
	PoseSpinAttack
	PoseSneaking
	PoseLongJumping
	PoseDying
	PoseCroaking
	PoseUsingTongue
	PoseSitting
	PoseRoaring
	PoseSniffing
	PoseEmerging
	PoseDigging
	PoseSliding
	PoseShooting
	PoseInhaling
)

const (
	TicksPerSecond = 20
	TicksPerDay    = 24000
)

var gameModeNames = [4]string{"SURVIVAL", "CREATIVE", "ADVENTURE", "SPECTATOR"}

func GameModeString(mode int) string {
	if mode >= 0 && mode < len(gameModeNames) {
		return gameModeNames[mode]
	}
	return "UNKNOWN"
}

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
	ActionLeaveBed             PlayerCommand = 0
	ActionStartSprinting       PlayerCommand = 1
	ActionStopSprinting        PlayerCommand = 2
	ActionStartJumpWithHorse   PlayerCommand = 3
	ActionStopJumpWithHorse    PlayerCommand = 4
	ActionOpenVehicleInventory PlayerCommand = 5
	ActionFlyingWithElytra     PlayerCommand = 6
)

const (
	InputForward PlayerInput = 1 << 0
	InputBack    PlayerInput = 1 << 1
	InputLeft    PlayerInput = 1 << 2
	InputRight   PlayerInput = 1 << 3
	InputJump    PlayerInput = 1 << 4
	InputSneak   PlayerInput = 1 << 5
	InputSprint  PlayerInput = 1 << 6
)

const (
	MaxQuantizedValue = 32766.0
)

var ServerDataPacks = NewPrefixedArray[DataPackIdentifier, *DataPackIdentifier]([]DataPackIdentifier{
	{
		Namespace: "minecraft",
		ID:        "core",
		Version:   mcdata.GameVersion,
	},
})

func (v VarInt) Len() int {
	val := uint32(v)

	if v < 0 {
		return 5
	}
	n := 1
	for val >= 0x80 {
		val >>= 7
		n++
	}
	return n
}

func NewArray[T any, PT FieldPtr[T]](size uint32) Array[T, PT] {
	return Array[T, PT]{
		Slice: make([]T, size),
	}
}

// NewPrefixedArray wraps an existing slice in a PrefixedArray.
func NewPrefixedArray[T any, PT FieldPtr[T]](slice []T) PrefixedArray[T, PT] {
	return PrefixedArray[T, PT]{Slice: slice}
}

// MapToPrefixedArray converts a slice of one type to a PrefixedArray of another type using a conversion function.
// ex: convert []byte to PrefixedArray[Byte]
func MapToPrefixedArray[E any, PE FieldPtr[E], S any](slice []S, convert func(S) E) PrefixedArray[E, PE] {
	if slice == nil {
		return PrefixedArray[E, PE]{}
	}
	newSlice := make([]E, len(slice))
	for i, v := range slice {
		newSlice[i] = convert(v)
	}
	return PrefixedArray[E, PE]{Slice: newSlice}
}

// CollectToPrefixedArray creates a new PrefixedArray from an iterator with a conversion function and filtering.
// ex: iter over connections map and return the player (when the connection has one)
func CollectToPrefixedArray[E any, PE FieldPtr[E], S any](seq iter.Seq[S], convert func(S) (E, bool)) PrefixedArray[E, PE] {
	var newSlice []E
	for v := range seq {
		if mapped, keep := convert(v); keep {
			newSlice = append(newSlice, mapped)
		}
	}
	return PrefixedArray[E, PE]{Slice: newSlice}
}

// MapToSlice converts a PrefixedArray to a regular slice using a conversion function.
// ex: convert PrefixedArray[Byte] to []byte
func MapToSlice[E any, PE FieldPtr[E], T any](p PrefixedArray[E, PE], convert func(E) T) []T {
	if p.Slice == nil {
		return nil
	}
	dst := make([]T, len(p.Slice))
	for i, v := range p.Slice {
		dst[i] = convert(v)
	}
	return dst
}

func NewPrefixedOptional[T any, PT FieldPtr[T]](val *T) PrefixedOptional[T, PT] {
	return PrefixedOptional[T, PT]{
		Has:   true,
		Value: val,
	}
}

func (b *BitSet) Set(i int, value bool) {
	data := b.Slice
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
	data := b.Slice
	idx := i / 64
	off := uint(i % 64)
	if idx >= len(data) {
		return false
	}
	return (data[idx] & (1 << off)) != 0
}

func (P *PrefixedOptional[T, PT]) Set(value PT) {
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

func NewDataArray(bitsPerEntry int, size int) *DataArray {
	valuesPerLong := 64 / bitsPerEntry
	longCount := (size + valuesPerLong - 1) / valuesPerLong

	return &DataArray{
		Slice:        make([]uint64, longCount),
		BitsPerEntry: bitsPerEntry,
		Mask:         (1 << bitsPerEntry) - 1,
		Size:         size,
	}
}

func (D *DataArray) Set(index int, value int) {
	if index < 0 || index >= D.Size {
		return
	}

	valuesPerLong := 64 / D.BitsPerEntry
	cellIndex := index / valuesPerLong
	bitIndex := (index % valuesPerLong) * D.BitsPerEntry

	D.Slice[cellIndex] = (D.Slice[cellIndex] &^ (D.Mask << bitIndex)) | (uint64(value) & D.Mask << bitIndex)
}

func (D *DataArray) Get(index int) int {
	if index < 0 || index >= D.Size {
		return 0
	}

	valuesPerLong := 64 / D.BitsPerEntry
	cellIndex := index / valuesPerLong
	bitIndex := (index % valuesPerLong) * D.BitsPerEntry

	return int((D.Slice[cellIndex] >> bitIndex) & D.Mask)
}

func (pm *PreviousMessages) Add(entry PreviousMessage) {
	pm.start = (pm.start - 1 + len(pm.entries)) % len(pm.entries)
	pm.entries[pm.start] = entry
	if pm.count < len(pm.entries) {
		pm.count++
	}
}

func (pm *PreviousMessages) Get(i int) PreviousMessage {
	return pm.entries[(pm.start+i)%len(pm.entries)]
}

func (pm *PreviousMessages) Len() int {
	return pm.count
}

func pack(value float64) int64 {
	return int64(math.Round((value*0.5 + 0.5) * MaxQuantizedValue))
}

func unpack(value uint64) float64 {
	v := float64(value & 32767)
	v = math.Min(v, MaxQuantizedValue)

	return v*2.0/MaxQuantizedValue - 1.0
}

func GetStateName(state State) string {
	switch state {
	case StateHandshake:
		return "Handshake"
	case StateStatus:
		return "Status"
	case StateLogin:
		return "Login"
	case StateConfiguration:
		return "Configuration"
	case StatePlay:
		return "Play"
	default:
		return "Unknown"
	}
}
