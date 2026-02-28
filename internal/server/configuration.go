package server

import (
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func (c *Connection) HandleClientKnownPacks(pkt *packet.Packet) {
	var knownPacks mc.PrefixedArray[mc.DataPackIdentifier]

	if err := pkt.Decode(&knownPacks); err != nil {
		logger.Error("Error decoding clientKnownPacks packet: %v", err)
		return
	}

	for _, registryData := range mc.RegistriesData {
		_ = pkt.ResetWith(packet.ConfigurationClientboundRegistryData, &registryData)
		_ = pkt.Send(c.Conn, c.CompressionThreshold)
	}

	// todo: send the update tags (optional but cause enchantment registry to not work)
	_ = pkt.ResetWith(packet.ConfigurationClientboundFinishConfiguration)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}

// todo: we should move packet sent to methods
func (c *Connection) HandleFinishConfigurationAck(pkt *packet.Packet) {
	// order: https://minecraft.wiki/w/Java_Edition_protocol/FAQ#What's_the_normal_login_sequence_for_a_client?
	c.Server.World.AddPlayer(c.Player)
	c.State = mc.StatePlay
	dimensionsName := []mc.String{"minecraft:overworld", "minecraft:the_nether", "minecraft:the_end"}

	_ = pkt.ResetWith(
		packet.PlayClientboundLogin,
		mc.Int(c.Player.EntityID),
		mc.Boolean(c.Server.Properties.Hardcore),
		mc.NewPrefixedArray(&dimensionsName),
		mc.VarInt(c.Server.Properties.MaxPlayers),
		mc.VarInt(c.Server.Properties.ViewDistance),
		mc.VarInt(c.Server.Properties.SimulationDistance),
		mc.Boolean(false),
		mc.Boolean(true),
		mc.Boolean(false),
		// todo: get the correct dimension type and name from player
		mc.VarInt(0),
		mc.String("minecraft:overworld"),
		// todo: hash world seed
		mc.Long(1),
		mc.UnsignedByte(c.Player.GameMode),
		mc.Byte(c.Player.PreviousGameMode),
		mc.Boolean(false),
		mc.Boolean(false),
		// todo: get the correct value
		mc.Boolean(false),
		mc.VarInt(100),
		mc.VarInt(64),
		mc.Boolean(false),
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	_ = pkt.ResetWith(packet.PlayClientboundSetHeldSlot, mc.VarInt(c.Player.SelectedItemSlot))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	c.Server.sendCommands(c)

	_ = pkt.ResetWith(
		packet.PlayClientboundPlayerPosition,
		mc.VarInt(0),
		mc.Double(c.Player.Pos[0]),
		mc.Double(c.Player.Pos[1]),
		mc.Double(c.Player.Pos[2]),
		// todo: replace with velocity
		mc.Double(c.Player.Motion[0]),
		mc.Double(c.Player.Motion[1]),
		mc.Double(c.Player.Motion[2]),
		mc.Float(c.Player.Rot[0]),
		mc.Float(c.Player.Rot[1]),
		mc.Int(0),
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	// todo: all the following packet must be sent in response of the Confirm Teleportation packet sent by the client after the previous Sync position packet

	c.syncMovement(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], true, true)

	players := []*entities.Player{c.Player}
	var allPlayers []*entities.Player
	for _, p := range c.Server.World.Players {
		allPlayers = append(allPlayers, p)
	}

	// todo: should also send gamemode
	actions := mc.ActionAddPlayer | mc.ActionUpdateListed
	pkt1, _ := packet.BuildPlayerInfoUpdatePacket(actions, players)
	c.Server.Broadcaster.Broadcast(pkt1, systems.NotSender(c))
	pkt1, _ = packet.BuildPlayerInfoUpdatePacket(actions, allPlayers)
	_ = pkt1.Send(c.Conn, c.CompressionThreshold)

	_ = pkt.ResetWith(
		packet.PlayClientboundSetTime,
		mc.Long(c.Server.World.Time),
		mc.Long(c.Server.World.DayTime),
		mc.Boolean(true),
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	_ = pkt.ResetWith(
		packet.PlayClientboundGameEvent,
		mc.UnsignedByte(13),
		mc.Float(0.0),
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	cx := int(math.Floor(c.Player.Pos[0] / 16.0))
	cz := int(math.Floor(c.Player.Pos[2] / 16.0))
	_ = pkt.ResetWith(packet.PlayClientboundSetChunkCacheCenter, mc.VarInt(cx), mc.VarInt(cz))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	dimension := c.Server.World.Dimensions["minecraft:overworld"]
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			pos := mc.ChunkPos{X: x, Z: z}
			chunk := dimension.GetChunk(x, z)

			_ = pkt.ResetWith(packet.PlayClientboundLevelChunkWithLight, chunk)
			_ = pkt.Send(c.Conn, c.CompressionThreshold)
			c.LoadedChunks[pos] = struct{}{}
		}
	}
	c.Player.Movement.LastChunkX = cx
	c.Player.Movement.LastChunkZ = cz

	// todo: following packets must be sent in response of the Player loaded packet

	// todo: send player inventory, rework inventory system

	velocity := mc.LpVec3{
		X: 0,
		Y: 0,
		Z: 0,
	}
	zeroAngle := mc.Angle(0)
	data := mc.VarInt(0)
	eID2 := mc.VarInt(c.Player.EntityID)
	uuid := mc.UUID(c.Player.UUID)
	// spawn newly connected player
	pkt, _ = packet.NewPacket(
		packet.PlayClientboundAddEntity,
		&eID2,
		&uuid,
		mc.VarInt(mcdata.EntityPlayer),
		mc.Double(c.Player.Pos[0]),
		mc.Double(c.Player.Pos[1]),
		mc.Double(c.Player.Pos[2]),
		&velocity,
		&zeroAngle,
		&zeroAngle,
		&zeroAngle,
		&data,
	)
	c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
	c.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player != nil && conn.Player.UUID != c.Player.UUID {
			uuid := mc.UUID(conn.Player.UUID)

			pkt, _ := packet.NewPacket(
				packet.PlayClientboundAddEntity,
				mc.VarInt(conn.Player.EntityID),
				&uuid,
				mc.VarInt(mcdata.EntityPlayer),
				mc.Double(conn.Player.Pos[0]),
				mc.Double(conn.Player.Pos[1]),
				mc.Double(conn.Player.Pos[2]),
				&velocity,
				&zeroAngle,
				&zeroAngle,
				&zeroAngle,
				&data,
			)

			_ = pkt.Send(c.Conn, c.CompressionThreshold)
		}
		return true
	})

	joinMessage := tc.Translatable(
		mcdata.MultiplayerPlayerJoined,
		tc.PlayerName(string(c.Player.Name)),
	).SetColor(tc.ColorYellow)
	pkt, _ = packet.NewPacket(packet.PlayClientboundSystemChat, joinMessage, mc.Boolean(false))
	c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}

func (s *Server) sendCommands(c *Connection) {
	flattenGraph, idMap := s.Commander.FlattenGraph()
	pkt, _ := packet.NewPacket(packet.PlayClientboundCommands, mc.VarInt(len(flattenGraph)))

	for _, node := range flattenGraph {
		flags := node.GetFlags()

		_ = pkt.Encode(mc.Byte(flags), mc.VarInt(len(node.Children)))
		for _, node := range node.Children {
			_ = pkt.Encode(mc.VarInt(idMap[node]))
		}

		if node.Redirect != nil {
			_ = pkt.Encode(mc.VarInt(idMap[node.Redirect]))
		}
		if node.Kind == commander.LiteralNode || node.Kind == commander.ArgumentNode {
			_ = pkt.Encode(mc.String(node.Name))

			if node.Kind == commander.ArgumentNode {
				_ = pkt.Encode(mc.VarInt(node.Parser.ID()))
				_, _ = node.Parser.WriteTo(pkt.Buffer)
			}
		}

		// todo: encode suggestion type
	}
	_ = pkt.Encode(mc.VarInt(0))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	pkt.Free()
}

func (c *Connection) HandleClientInformation(pkt *packet.Packet) {
	// NOTE: this packet can be sent in configuration and play state
	var information mc.PlayerInformation
	shouldUpdateChunks := false

	if err := pkt.Decode(&information); err != nil {
		logger.Error("Error decoding clientInformation packet: %v", err)
		return
	}

	switch {
	case information.ViewDistance < 2:
		information.ViewDistance = 2
	case int(information.ViewDistance) > c.Server.Properties.ViewDistance:
		information.ViewDistance = mc.Byte(c.Server.Properties.ViewDistance)
	}

	if c.State == mc.StatePlay {
		if information.ViewDistance != c.Player.Information.ViewDistance {
			shouldUpdateChunks = true
		}
	}
	c.Player.Information = information

	if shouldUpdateChunks {
		pkt, _ := packet.NewPacket(packet.PlayClientboundSetChunkCacheRadius, mc.VarInt(information.ViewDistance))

		c.Send(pkt)
		c.updateChunkView(true)
	}
}
