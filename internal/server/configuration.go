package server

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func (c *Connection) HandleServerboundKnownPacks(knownPacks *mc.PrefixedArray[mc.DataPackIdentifier, *mc.DataPackIdentifier]) {
	for _, registryData := range mc.RegistriesData {
		pkt, _ := packet.NewPacket(packet.ConfigurationClientboundRegistryData, &registryData)
		c.Send(pkt)
	}

	// todo: send the update tags (optional but cause enchantment registry to not work)
	pkt, _ := packet.NewPacket(packet.ConfigurationClientboundFinishConfiguration)
	c.Send(pkt)
}

// HandleAcknowledgeFinishConfiguration todo: we should move packet sent to methods
func (c *Connection) HandleAcknowledgeFinishConfiguration(_ *packet.InboundPacket) {
	// order: https://minecraft.wiki/w/Java_Edition_protocol/FAQ#What's_the_normal_login_sequence_for_a_client?
	// todo: move this to login -> avoid slot stealing and potential conflict
	if err := c.Server.World.AddPlayer(c.Player, "minecraft:overworld"); err != nil {
		logger.Error("Failed to spawn player %s: %v", logger.Identity(c.Player.Name), err)
		c.close()
		return
	}
	logger.Info("%s[/%s] logged in with entity id %s at (%s)",
		logger.Identity(c.Player.Name),
		logger.Network(c.Conn.RemoteAddr()),
		logger.Value(c.Player.EntityID),
		logger.Value(fmt.Sprintf("%f, %f, %f", c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2])),
	)
	c.State = mc.StatePlay
	c.Server.ConnectionsByEID[c.Player.EntityID] = c
	dimensionsName := []mc.Identifier{"overworld", "the_nether", "the_end"}

	out, _ := packet.NewPacket(0)
	defer out.Free()

	_ = out.ResetWith(
		packet.PlayClientboundLogin,
		mc.Int(c.Player.EntityID),
		mc.Boolean(c.Server.Properties.Hardcore),
		mc.NewPrefixedArray[mc.Identifier, *mc.Identifier](dimensionsName),
		mc.VarInt(c.Server.Properties.MaxPlayers),
		mc.VarInt(c.Server.Properties.ViewDistance),
		mc.VarInt(c.Server.Properties.SimulationDistance),
		mc.Boolean(false),
		mc.Boolean(true),
		mc.Boolean(false),
		// todo: get the correct dimension type and name from player
		mc.VarInt(0),
		mc.Identifier("overworld"),
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
		mc.Boolean(c.Server.EnforceSecureChat), // apparently, always false in offline mode
	)
	_ = out.Send(c.Conn, c.CompressionThreshold)

	_ = out.ResetWith(packet.PlayClientboundSetHeldSlot, mc.VarInt(c.Player.SelectedItemSlot))
	_ = out.Send(c.Conn, c.CompressionThreshold)

	c.Server.sendCommands(c)

	_ = out.ResetWith(
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
	_ = out.Send(c.Conn, c.CompressionThreshold)

	// todo: all the following packet must be sent in response of the Confirm Teleportation packet sent by the client after the previous Sync position packet

	c.syncMovement(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], true, true)

	me := []*entities.Player{c.Player}
	allPlayers := c.Server.World.Players()

	// todo: should also send gamemode
	actions := mc.ActionAddPlayer | mc.ActionUpdateListed
	pkt1, _ := buildPlayerInfoUpdatePacket(actions, me)
	c.Server.BroadcastOthers(c, pkt1)
	pkt1, _ = buildPlayerInfoUpdatePacket(actions|mc.ActionInitializeChat, allPlayers)
	_ = pkt1.Send(c.Conn, c.CompressionThreshold)

	_ = out.ResetWith(
		packet.PlayClientboundSetTime,
		mc.Long(c.Server.World.Time),
		mc.Long(c.Server.World.DayTime),
		mc.Boolean(true),
	)
	_ = out.Send(c.Conn, c.CompressionThreshold)

	_ = out.ResetWith(
		packet.PlayClientboundGameEvent,
		mc.UnsignedByte(13),
		mc.Float(0.0),
	)
	_ = out.Send(c.Conn, c.CompressionThreshold)

	cx, cz := world.GetChunkPosition(c.Player.Pos[0], c.Player.Pos[2])
	_ = out.ResetWith(packet.PlayClientboundSetChunkCacheCenter, mc.VarInt(cx), mc.VarInt(cz))
	_ = out.Send(c.Conn, c.CompressionThreshold)

	dimension := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			pos := mc.ChunkPos{X: x, Z: z}
			chunk := dimension.GetChunk(x, z)

			_ = out.ResetWith(packet.PlayClientboundLevelChunkWithLight, chunk)
			_ = out.Send(c.Conn, c.CompressionThreshold)

			chunk.Watchers[c.Player.EntityID] = struct{}{}
			c.Player.Movement.VisibleChunks[pos] = struct{}{}
		}
	}
	c.Player.Movement.LastChunkX = cx
	c.Player.Movement.LastChunkZ = cz

	// todo: following packets must be sent in response of the Player loaded packet
	// todo: send player inventory, rework inventory system

	// spawn newly connected player
	pkt, _ := packet.NewPacket(
		packet.PlayClientboundAddEntity,
		mc.VarInt(c.Player.EntityID),
		mc.UUID(c.Player.UUID),
		mc.VarInt(mcdata.EntityPlayer),
		mc.Double(c.Player.Pos[0]),
		mc.Double(c.Player.Pos[1]),
		mc.Double(c.Player.Pos[2]),
		mc.LpVec3{},
		mc.Angle(0),
		mc.Angle(0),
		mc.Angle(0),
		mc.VarInt(0),
	)
	c.Server.BroadcastViewers(c, pkt)
	for _, player := range c.Server.World.PlayersInChunkRadius("minecraft:overworld", cx, cz, loadRadius) {
		if player.UUID == c.Player.UUID {
			continue
		}

		pkt, _ := packet.NewPacket(
			packet.PlayClientboundAddEntity,
			mc.VarInt(player.EntityID),
			mc.UUID(player.UUID),
			mc.VarInt(mcdata.EntityPlayer),
			mc.Double(player.Pos[0]),
			mc.Double(player.Pos[1]),
			mc.Double(player.Pos[2]),
			mc.LpVec3{},
			mc.Angle(0),
			mc.Angle(0),
			mc.Angle(0),
			mc.VarInt(0),
		)

		_ = pkt.Send(c.Conn, c.CompressionThreshold)
	}

	joinMessage := tc.Translatable(
		mcdata.MultiplayerPlayerJoined,
		tc.PlayerName(c.Player.Name),
	).SetColor(tc.ColorYellow)
	pkt, _ = packet.NewPacket(packet.PlayClientboundSystemChat, joinMessage, mc.Boolean(false))
	c.Server.BroadcastOthers(c, pkt)
	logger.Component(logger.INFO, joinMessage)
	c.Server.ConnectionsByEID[c.Player.EntityID] = c
	c.State = mc.StatePlay
}

func (s *Server) sendCommands(c *Connection) {
	flattenGraph, idMap, filteredChildren := s.Commander.FlattenGraph(c.Player.PermissionLevel)
	pkt, _ := packet.NewPacket(packet.PlayClientboundCommands, mc.VarInt(len(flattenGraph)))

	for _, node := range flattenGraph {
		flags := node.GetFlags()
		children := filteredChildren[node]

		_ = pkt.Encode(mc.Byte(flags), mc.VarInt(len(children)))
		for _, child := range children {
			_ = pkt.Encode(mc.VarInt(idMap[child]))
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
		if node.Suggestion != commander.SuggestNothing {
			_ = pkt.Encode(mc.String(node.Suggestion))
		}
	}
	_ = pkt.Encode(mc.VarInt(0))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	pkt.Free()
}

func (c *Connection) HandleClientInformation(information *mc.ClientInformation) {
	// NOTE: this packet can be sent in configuration and play state
	shouldUpdateChunks := false

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
	c.Player.Information = *information

	if shouldUpdateChunks {
		pkt, _ := packet.NewPacket(packet.PlayClientboundSetChunkCacheRadius, mc.VarInt(information.ViewDistance))

		c.Send(pkt)
		c.updateChunkView(true)
	}
}
