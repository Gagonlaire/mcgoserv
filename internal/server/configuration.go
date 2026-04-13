package server

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func (c *Connection) HandleServerboundKnownPacks(knownPacks *mc.PrefixedArray[mc.DataPackIdentifier, *mc.DataPackIdentifier]) {
	for _, registryData := range mc.RegistriesData {
		pkt := c.NewPacket(packet.ConfigurationClientboundRegistryData, &registryData)
		c.Send(pkt)
	}

	// todo: send the update tags (optional but cause enchantment registry to not work)
	pkt := c.NewPacket(packet.ConfigurationClientboundFinishConfiguration)
	c.Send(pkt)
}

// HandleAcknowledgeFinishConfiguration todo: we should move packet sent to methods
func (c *Connection) HandleAcknowledgeFinishConfiguration(_ *packet.InboundPacket) {
	// order: https://minecraft.wiki/w/Java_Edition_protocol/FAQ#What's_the_normal_login_sequence_for_a_client?
	// todo: move this to login -> avoid slot stealing and potential conflict
	if err := c.Server.World.AddPlayer(c.Player, "minecraft:overworld"); err != nil {
		logger.Error("Failed to spawn player %s: %v", logger.Identity(c.Player.Name), err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	c.Server.ConnectionsByEID.Store(c.Player.EntityID, c)
	c.State = mc.StatePlay
	logger.Debug("%s entering play state", c.Player.Name)
	logger.Info("%s[/%s] logged in with entity id %s at (%s)",
		logger.Identity(c.Player.Name),
		logger.Network(c.Conn.RemoteAddr()),
		logger.Value(c.Player.EntityID),
		logger.Value(fmt.Sprintf("%f, %f, %f", c.Player.Position[0], c.Player.Position[1], c.Player.Position[2])),
	)
	out := c.NewPacket(0)
	if out == nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	defer out.Free()

	// todo: get the correct dimension type and name from player
	// todo: hash world seed
	// todo: get the correct has death location value
	_ = out.ResetWith(packet.PlayClientboundLogin, &encoders.Login{
		EntityID:            mc.Int(c.Player.EntityID),
		IsHardcore:          mc.Boolean(c.Server.Config.Server.Hardcore),
		DimensionNames:      mc.NewPrefixedArray[mc.Identifier, *mc.Identifier]([]mc.Identifier{"overworld", "the_nether", "the_end"}),
		MaxPlayers:          mc.VarInt(c.Server.Config.Server.MaxPlayers),
		ViewDistance:        mc.VarInt(c.Server.Config.Performance.MaxViewDistance),
		SimulationDistance:  mc.VarInt(c.Server.Config.Performance.SimulationDistance),
		ReducedDebugInfo:    false,
		EnableRespawnScreen: true,
		DoLimitedCrafting:   false,
		DimensionType:       0,
		DimensionName:       "overworld",
		HashedSeed:          1,
		GameMode:            mc.UnsignedByte(c.Player.GameMode),
		PreviousGameMode:    mc.Byte(c.Player.PreviousGameMode),
		IsDebug:             false,
		IsFlat:              false,
		HasDeathLocation:    false,
		PortalCooldown:      100,
		SeaLevel:            64,
		EnforceSecureChat:   mc.Boolean(c.Server.EnforceSecureChat), // apparently, always false in offline mode
	})
	_ = out.Send(c.Conn, c.CompressionThreshold)

	_ = out.ResetWith(packet.PlayClientboundSetHeldSlot, mc.VarInt(c.Player.SelectedItemSlot))
	_ = out.Send(c.Conn, c.CompressionThreshold)

	if err := c.Server.SendCommands(c); err != nil {
		logger.Error("Player disconnected during configuration: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}

	_ = out.ResetWith(
		packet.PlayClientboundPlayerPosition,
		mc.VarInt(0),
		mc.NewCoordinate(c.Player.Position),
		mc.NewCoordinate(c.Player.Motion),
		mc.Float(c.Player.Rotation[0]),
		mc.Float(c.Player.Rotation[1]),
		mc.Int(0),
	)
	_ = out.Send(c.Conn, c.CompressionThreshold)

	// todo: all the following packet must be sent in response of the Confirm Teleportation packet sent by the client after the previous Sync position packet

	c.syncMovement(c.Player.Position[0], c.Player.Position[1], c.Player.Position[2], true, true)

	me := []*entities.Player{c.Player}
	allPlayers := c.Server.World.Players()

	// todo: should also send gamemode
	actions := mc.ListActionAddPlayer | mc.ListActionUpdateListed
	pkt1, _ := buildPlayerInfoUpdatePacket(actions, me)
	c.Server.BroadcastOthers(c, pkt1)
	pkt1, _ = buildPlayerInfoUpdatePacket(actions|mc.ListActionInitializeChat, allPlayers)
	if pkt1 != nil {
		_ = pkt1.Send(c.Conn, c.CompressionThreshold)
	}

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

	cx, cz := world.GetChunkPosition(c.Player.Position[0], c.Player.Position[2])
	_ = out.ResetWith(packet.PlayClientboundSetChunkCacheCenter, mc.VarInt(cx), mc.VarInt(cz))
	_ = out.Send(c.Conn, c.CompressionThreshold)

	dimension := c.Server.World.GetEntityDimension(c.Player)
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	logger.Debug("Sending initial chunks to %s (center=[%d, %d], radius=%d)", c.Player.Name, cx, cz, loadRadius)
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

	if err := out.Err(); err != nil {
		logger.Error("Player disconnected during configuration: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}

	// todo: following packets must be sent in response of the Player loaded packet
	// todo: send player inventory, rework inventory system

	// spawn newly connected player
	pkt := c.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(c.Player))
	c.Server.BroadcastViewers(c, pkt)
	for player := range c.Server.World.PlayersInChunkRadius("minecraft:overworld", cx, cz, loadRadius) {
		if player.UUID == c.Player.UUID {
			continue
		}

		spawnPkt := c.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(player))
		if spawnPkt != nil {
			_ = spawnPkt.Send(c.Conn, c.CompressionThreshold)
			spawnPkt.Free()
		}
	}

	joinMessage := tc.Translatable(
		mcdata.MultiplayerPlayerJoined,
		tc.PlayerName(c.Player.Name),
	).SetColor(tc.ColorYellow)
	pkt = c.NewPacket(packet.PlayClientboundSystemChat, joinMessage, mc.Boolean(false))
	c.Server.BroadcastOthers(c, pkt)
	logger.Component(logger.INFO, joinMessage)
}

func (s *Server) SendCommands(c *Connection) error {
	flattenGraph, idMap, filteredChildren := s.Commander.FlattenGraph(c.Player.PermissionLevel)
	pkt := c.NewPacket(packet.PlayClientboundCommands, mc.VarInt(len(flattenGraph)))
	if pkt == nil {
		return fmt.Errorf("failed to create commands packet")
	}
	defer pkt.Free()

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
	return pkt.Err()
}

func (c *Connection) HandleClientInformation(information *mc.ClientInformation) {
	// NOTE: this packet can be sent in configuration and play state
	shouldUpdateChunks := false

	switch {
	case information.ViewDistance < 2:
		information.ViewDistance = 2
	case int(information.ViewDistance) > c.Server.Config.Performance.MaxViewDistance:
		information.ViewDistance = mc.Byte(c.Server.Config.Performance.MaxViewDistance)
	}

	if c.State == mc.StatePlay {
		if information.ViewDistance != c.Player.Information.ViewDistance {
			shouldUpdateChunks = true
		}
	}
	c.Player.Information = *information

	if shouldUpdateChunks {
		logger.Debug("%s changed view distance to %d", c.Player.Name, information.ViewDistance)
		pkt := c.NewPacket(packet.PlayClientboundSetChunkCacheRadius, mc.VarInt(information.ViewDistance))

		c.Send(pkt)
		c.updateChunkView(true)
	}
}
