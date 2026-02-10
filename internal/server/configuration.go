package server

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func (c *Connection) HandleClientKnownPacksPacket(pkt *packet.Packet) {
	var knownPacks mc.PrefixedArray[mc.DataPackIdentifier]

	if err := pkt.Decode(&knownPacks); err != nil {
		fmt.Println("Error decoding clientKnownPacks packet:", err)
		return
	}

	for _, registryData := range mc.RegistriesData {
		_ = pkt.ResetWith(packet.ConfigurationClientboundRegistryData, &registryData)
		_ = pkt.Send(c.Conn)
	}

	// todo: send the update tags (optional but cause enchantment registry to not work)
	_ = pkt.ResetWith(packet.ConfigurationClientboundFinishConfiguration)
	_ = pkt.Send(c.Conn)
}

// todo: we should move packet sent to methods
func (c *Connection) HandleFinishConfigurationAckPacket(pkt *packet.Packet) {
	c.State = mc.StatePlay
	c.server.World.AddPlayer(c.Player)
	// todo: check for entity id generation
	// todo: dimensionType should automatically send the good id
	// todo: create the optional type that should take a pointer to a value and is evaluated during the packet send/receive
	var (
		eID                 = c.Player.EntityID
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
		packet.PlayClientboundLogin,
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
	_ = pkt.Send(c.Conn)

	_ = pkt.ResetWith(
		packet.PlayClientboundSynchronizePlayerPosition,
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
	_ = pkt.Send(c.Conn)

	_ = pkt.ResetWith(
		packet.PlayClientboundGameEvent,
		&event,
		&value,
	)
	_ = pkt.Send(c.Conn)

	players := []*world.Player{c.Player}
	allPlayers := slices.Collect(maps.Values(c.server.World.Players))

	actions := mc.ActionAddPlayer | mc.ActionUpdateListed
	pkt1, _ := packet.BuildPlayerInfoUpdatePacket(actions, players)
	c.server.Broadcaster.Broadcast(pkt1, systems.NotSender(c))
	pkt1, _ = packet.BuildPlayerInfoUpdatePacket(actions, allPlayers)
	_ = pkt1.Send(c.Conn)

	worldAge := mc.Long(c.server.World.Time)
	timeOfDay := mc.Long(c.server.World.DayTime)
	timeOfDayIncreasing := mc.Boolean(true)
	_ = pkt.ResetWith(
		packet.PlayClientboundSetTime,
		&worldAge,
		&timeOfDay,
		&timeOfDayIncreasing,
	)
	_ = pkt.Send(c.Conn)

	for x := -10; x <= 10; x++ {
		for z := -10; z <= 10; z++ {
			// Create a chunk with random data for now
			chunk := mc.CreateChunk(x, z)
			_ = pkt.ResetWith(packet.PlayClientboundChunkDataAndUpdateLight, chunk)
			_ = pkt.Send(c.Conn)
		}
	}

	entityType := mc.VarInt(151)
	velocity := mc.LpVec3{
		X: 0,
		Y: 0,
		Z: 0,
	}
	zeroAngle := mc.Angle(0)
	data := mc.VarInt(0)
	eID2 := mc.VarInt(c.Player.EntityID)
	// spawn newly connected player
	pkt, _ = packet.NewPacket(
		packet.PlayClientboundSpawnEntity,
		&eID2,
		&c.Player.UUID,
		&entityType,
		&c.Player.X,
		&c.Player.Y,
		&c.Player.Z,
		&velocity,
		&zeroAngle,
		&zeroAngle,
		&zeroAngle,
		&data,
	)
	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
	c.server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player.UUID != c.Player.UUID {
			eID3 := mc.VarInt(conn.Player.EntityID)

			pkt, _ := packet.NewPacket(
				packet.PlayClientboundSpawnEntity,
				&eID3,
				&conn.Player.UUID,
				&entityType,
				&conn.Player.X,
				&conn.Player.Y,
				&conn.Player.Z,
				&velocity,
				&zeroAngle,
				&zeroAngle,
				&zeroAngle,
				&data,
			)

			_ = pkt.Send(c.Conn)
		}
		return true
	})
}
