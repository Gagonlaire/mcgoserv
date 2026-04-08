package mc

import (
	"crypto/rsa"
	"io"

	"github.com/google/uuid"
)

const (
	TicksPerSecond = 20
	TicksPerDay    = 24000
)

type State VarInt

const (
	StateHandshake State = iota
	StateStatus
	StateLogin
	StateConfiguration
	StatePlay
	StateMax
)

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

type PlayerAction VarInt

const (
	ActionStartDigging PlayerAction = iota
	ActionCancelDigging
	ActionFinishDigging
	ActionDropItemStack
	ActionDropItem
	ActionReleaseUseItem
	ActionSwapHand
)

// https://minecraft.wiki/w/Java_Edition_protocol/Packets#Player_Info_Update
type PlayerListAction UnsignedByte

const (
	ListActionAddPlayer          PlayerListAction = 1 << 0
	ListActionInitializeChat     PlayerListAction = 1 << 1
	ListActionUpdateGameMode     PlayerListAction = 1 << 2
	ListActionUpdateListed       PlayerListAction = 1 << 3
	ListActionUpdateLatency      PlayerListAction = 1 << 4
	ListActionUpdateDisplayName  PlayerListAction = 1 << 5
	ListActionUpdateListPriority PlayerListAction = 1 << 6
	ListActionUpdateHat          PlayerListAction = 1 << 7
)

type PlayerCommand VarInt

const (
	CommandLeaveBed PlayerCommand = iota
	CommandStartSprinting
	CommandStopSprinting
	CommandStartJumpWithHorse
	CommandStopJumpWithHorse
	CommandOpenVehicleInventory
	CommandFlyingWithElytra
)

type PlayerInput UnsignedByte

const (
	InputForward PlayerInput = 1 << 0
	InputBack    PlayerInput = 1 << 1
	InputLeft    PlayerInput = 1 << 2
	InputRight   PlayerInput = 1 << 3
	InputJump    PlayerInput = 1 << 4
	InputSneak   PlayerInput = 1 << 5
	InputSprint  PlayerInput = 1 << 6
)

type TeleportationFlags Int

const (
	TeleportationFlagsRelativeX         TeleportationFlags = 1 << 0
	TeleportationFlagsRelativeY         TeleportationFlags = 1 << 1
	TeleportationFlagsRelativeZ         TeleportationFlags = 1 << 2
	TeleportationFlagsRelativeYaw       TeleportationFlags = 1 << 3
	TeleportationFlagsRelativePitch     TeleportationFlags = 1 << 4
	TeleportationFlagsRelativeVelocityX TeleportationFlags = 1 << 5
	TeleportationFlagsRelativeVelocityY TeleportationFlags = 1 << 6
	TeleportationFlagsRelativeVelocityZ TeleportationFlags = 1 << 7
	TeleportationFlagsRotateVelocity    TeleportationFlags = 1 << 8
)

var gameModeNames = [4]string{"SURVIVAL", "CREATIVE", "ADVENTURE", "SPECTATOR"}

func GameModeString(mode int) string {
	if mode >= 0 && mode < len(gameModeNames) {
		return gameModeNames[mode]
	}
	return "UNKNOWN"
}

type ProfileProperty struct {
	Name      string
	Value     string
	Signature string
}

func (p *ProfileProperty) ReadFrom(_ io.Reader) (int64, error) {
	panic("Not implemented")
}

func (p ProfileProperty) WriteTo(w io.Writer) (n int64, err error) {
	nn, err := String(p.Name).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	nn, err = String(p.Value).WriteTo(w)
	n += nn
	if err != nil {
		return n, err
	}
	if p.Signature != "" {
		nn, err = Boolean(true).WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
		nn, err = String(p.Signature).WriteTo(w)
		n += nn
		return n, err
	}
	nn, err = Boolean(false).WriteTo(w)
	n += nn
	return n, err
}

//field:encode mode=both
type ClientInformation struct {
	Locale              String16
	ViewDistance        Byte
	ChatMode            VarInt
	ChatColors          Boolean // Unused by vanilla server
	DisplayedSkinParts  UnsignedByte
	MainHand            VarInt
	EnableTextFiltering Boolean
	AllowServerListings Boolean
	ParticleStatus      VarInt
}

type PreviousMessage struct {
	Signature []byte
	MessageID int32
}

type PreviousMessages struct {
	entries [20]PreviousMessage
	start   int
	count   int
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

type ChatSession struct {
	PublicKey        *rsa.PublicKey
	KeySignature     []byte
	PreviousMessages PreviousMessages
	ExpiresAt        int64
	Index            int32
	GlobalIndex      int32
	LastSeenCount    int32
	ID               uuid.UUID
	Signed           bool
}
