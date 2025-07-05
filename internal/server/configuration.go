package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleClientKnownPacksPacket(conn *Connection, pkt *packet.Packet) {
	var knownPacks mc.PrefixedArray[mc.DataPackIdentifier]

	if err := pkt.Decode(&knownPacks); err != nil {
		fmt.Println("Error decoding clientKnownPacks packet:", err)
		return
	}

	for _, registryData := range mc.RegistriesData {
		_ = pkt.ResetWith(0x07, &registryData)
		_ = pkt.Send(conn.Conn)
	}

	// todo: send the update tags (optional but cause enchantment registry to not work)
	_ = pkt.ResetWith(0x03)
	_ = pkt.Send(conn.Conn)
}

func HandleFinishConfigurationAckPacket(conn *Connection, pkt *packet.Packet) {
	conn.State = StatePlay
	// todo: check for entity id generation
	// todo: dimensionType should automatically send the good id
	// todo: create the optional type that should take a pointer to a value and is evaluated during the packet send/receive
	var (
		eID                 = mc.Int(0)
		isHardcore          = mc.Boolean(false)
		dimensionNames      = []mc.String{"minecraft:overworld", "minecraft:the_nether", "minecraft:the_end"}
		maxPlayers          = mc.VarInt(20)
		viewDistance        = mc.VarInt(32)
		simulationDistance  = mc.VarInt(32)
		reduceDebugInfo     = mc.Boolean(false)
		enableRespawnScreen = mc.Boolean(true)
		doLimitedCrafting   = mc.Boolean(false)
		dimensionType       = mc.VarInt(0)
		dimensionName       = mc.String("minecraft:overworld")
		hashedSeed          = mc.Long(1)
		gameMode            = mc.UnsignedByte(0)
		previousGameMode    = mc.Byte(-1)
		isDebug             = mc.Boolean(false)
		isFlat              = mc.Boolean(false)
		hasDeathLocation    = mc.Boolean(false)
		portalCooldown      = mc.VarInt(100)
		seaLevel            = mc.VarInt(64)
		enforceSecureChat   = mc.Boolean(false)
	)

	var (
		teleportId = mc.VarInt(0)
		x          = mc.Double(0.0)
		y          = mc.Double(80)
		z          = mc.Double(0.0)
		velocityX  = mc.Double(0.0)
		velocityY  = mc.Double(0.0)
		velocityZ  = mc.Double(0.0)
		yaw        = mc.Float(0.0)
		pitch      = mc.Float(0.0)
		flags      = mc.Int(0)
	)

	var (
		event = mc.UnsignedByte(13)
		value = mc.Float(0.0)
	)

	_ = pkt.ResetWith(
		0x2B,
		&eID,
		&isHardcore,
		mc.NewPrefixedArray(&dimensionNames),
		&maxPlayers,
		&viewDistance,
		&simulationDistance,
		&reduceDebugInfo,
		&enableRespawnScreen,
		&doLimitedCrafting,
		&dimensionType,
		&dimensionName,
		&hashedSeed,
		&gameMode,
		&previousGameMode,
		&isDebug,
		&isFlat,
		&hasDeathLocation,
		&portalCooldown,
		&seaLevel,
		&enforceSecureChat,
	)
	_ = pkt.Send(conn.Conn)
	_ = pkt.ResetWith(
		0x41,
		&teleportId,
		&x,
		&y,
		&z,
		&velocityX,
		&velocityY,
		&velocityZ,
		&yaw,
		&pitch,
		&flags,
	)
	_ = pkt.Send(conn.Conn)
	_ = pkt.ResetWith(
		0x22,
		&event,
		&value,
	)
	_ = pkt.Send(conn.Conn)

	for x := -10; x <= 10; x++ {
		for z := -10; z <= 10; z++ {
			// Create a chunk with random data for now
			chunk := mc.CreateChunk(x, z)
			_ = pkt.ResetWith(0x27, chunk)
			_ = pkt.Send(conn.Conn)
		}
	}
}
