package server

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func (c *Connection) HandleServerboundKnownPacks(knownPacks *proto.PrefixedArray[mc.DataPackIdentifier, *mc.DataPackIdentifier]) {
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
	if err := c.Server.SpawnPlayer(c.Player, "minecraft:overworld"); err != nil {
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
	// todo: get the correct dimension type and name from player
	// todo: hash world seed
	// todo: get the correct has death location value
	_ = c.SendSync(c.NewPacket(packet.PlayClientboundLogin, &encoders.Login{
		EntityID:            proto.Int(c.Player.EntityID),
		IsHardcore:          proto.Boolean(c.Server.Config.Server.Hardcore),
		DimensionNames:      proto.NewPrefixedArray[proto.Identifier, *proto.Identifier]([]proto.Identifier{"overworld", "the_nether", "the_end"}),
		MaxPlayers:          proto.VarInt(c.Server.Config.Server.MaxPlayers),
		ViewDistance:        proto.VarInt(c.Server.Config.Performance.MaxViewDistance),
		SimulationDistance:  proto.VarInt(c.Server.Config.Performance.SimulationDistance),
		ReducedDebugInfo:    false,
		EnableRespawnScreen: true,
		DoLimitedCrafting:   false,
		DimensionType:       0,
		DimensionName:       "overworld",
		HashedSeed:          1,
		GameMode:            proto.UnsignedByte(c.Player.GameMode),
		PreviousGameMode:    proto.Byte(c.Player.PreviousGameMode),
		IsDebug:             false,
		IsFlat:              false,
		HasDeathLocation:    false,
		PortalCooldown:      100,
		SeaLevel:            64,
		EnforceSecureChat:   proto.Boolean(c.Server.EnforceSecureChat), // apparently, always false in offline mode
	}))

	_ = c.SendSync(c.NewPacket(packet.PlayClientboundSetHeldSlot, proto.VarInt(c.Player.Inventory.SelectedHotbar)))

	if err := c.Server.SendCommands(c); err != nil {
		logger.Error("Player disconnected during configuration: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}

	_ = c.SendSync(c.NewPacket(
		packet.PlayClientboundPlayerPosition,
		proto.VarInt(0),
		proto.NewCoordinate(c.Player.Position),
		proto.NewCoordinate(c.Player.Motion),
		proto.Float(c.Player.Rotation[0]),
		proto.Float(c.Player.Rotation[1]),
		proto.Int(0),
	))

	// todo: all the following packet must be sent in response of the Confirm Teleportation packet sent by the client after the previous Sync position packet

	c.syncMovement(c.Player.Position[0], c.Player.Position[1], c.Player.Position[2], true, true)

	me := []*entity.Player{c.Player}
	allPlayers := c.Server.World.Players()

	// todo: should also send gamemode
	actions := mc.ListActionAddPlayer | mc.ListActionUpdateListed
	pkt1, _ := buildPlayerInfoUpdatePacket(actions, me)
	c.Server.BroadcastOthers(c, pkt1)
	c.Server.broadcastSpawn(c.Player)
	pkt1, _ = buildPlayerInfoUpdatePacket(actions|mc.ListActionInitializeChat, allPlayers)
	_ = c.SendSync(pkt1)

	_ = c.SendSync(c.NewPacket(
		packet.PlayClientboundSetTime,
		proto.Long(c.Server.World.Time),
		proto.Long(c.Server.World.DayTime),
		proto.Boolean(true),
	))

	_ = c.SendSync(c.NewPacket(
		packet.PlayClientboundGameEvent,
		proto.UnsignedByte(13),
		proto.Float(0.0),
	))

	cx, cz := world.GetChunkPosition(c.Player.Position[0], c.Player.Position[2])
	_ = c.SendSync(c.NewPacket(packet.PlayClientboundSetChunkCacheCenter, proto.VarInt(cx), proto.VarInt(cz)))

	dimension := c.Server.World.GetEntityDimension(c.Player)
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	logger.Debug("Sending initial chunks to %s (center=[%d, %d], radius=%d)", c.Player.Name, cx, cz, loadRadius)
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			pos := mc.ChunkPos{X: x, Z: z}
			chunk := dimension.GetChunk(x, z)

			_ = c.SendSync(c.NewPacket(packet.PlayClientboundLevelChunkWithLight, encoders.NewChunkData(chunk)))

			chunk.Watchers[c.Player.EntityID] = struct{}{}
			c.Player.Movement.VisibleChunks[pos] = struct{}{}
			c.SendChunkEntities(chunk)
		}
	}
	c.Player.Movement.LastChunkX = cx
	c.Player.Movement.LastChunkZ = cz

	// todo: following packets must be sent in response of the Player loaded packet
	// todo: send player inventory, rework inventory system

	joinMessage := tc.Translatable(
		mcdata.MultiplayerPlayerJoined,
		tc.PlayerName(c.Player.Name),
	).SetColor(tc.ColorYellow)
	pkt := c.NewPacket(packet.PlayClientboundSystemChat, joinMessage, proto.Boolean(false))
	c.Server.BroadcastOthers(c, pkt)
	logger.Component(logger.INFO, joinMessage)
}

func (s *Server) SendCommands(c *Connection) error {
	flattenGraph, idMap, filteredChildren := s.Commander.FlattenGraph(c.Player.PermissionLevel)
	pkt := c.NewPacket(packet.PlayClientboundCommands, proto.VarInt(len(flattenGraph)))
	if pkt == nil {
		return fmt.Errorf("failed to create commands packet")
	}

	for _, node := range flattenGraph {
		flags := node.GetFlags()
		children := filteredChildren[node]

		_ = pkt.Encode(proto.Byte(flags), proto.VarInt(len(children)))
		for _, child := range children {
			_ = pkt.Encode(proto.VarInt(idMap[child]))
		}

		if node.Redirect != nil {
			_ = pkt.Encode(proto.VarInt(idMap[node.Redirect]))
		}
		if node.Kind == commander.LiteralNode || node.Kind == commander.ArgumentNode {
			_ = pkt.Encode(proto.String(node.Name))

			if node.Kind == commander.ArgumentNode {
				_ = pkt.Encode(proto.VarInt(node.Parser.ID()))
				_, _ = node.Parser.WriteTo(pkt.Buffer)
			}
		}
		if node.Suggestion != commander.SuggestNothing {
			_ = pkt.Encode(proto.String(node.Suggestion))
		}
	}
	_ = pkt.Encode(proto.VarInt(0))
	err := pkt.Err()
	_ = c.SendSync(pkt)
	return err
}

func (c *Connection) HandleClientInformation(information *mc.ClientInformation) {
	// NOTE: this packet can be sent in configuration and play state
	shouldUpdateChunks := false

	switch {
	case information.ViewDistance < 2:
		information.ViewDistance = 2
	case int(information.ViewDistance) > c.Server.Config.Performance.MaxViewDistance:
		information.ViewDistance = proto.Byte(c.Server.Config.Performance.MaxViewDistance)
	}

	if c.State == mc.StatePlay {
		if information.ViewDistance != c.Player.Information.ViewDistance {
			shouldUpdateChunks = true
		}
	}
	c.Player.Information = *information

	if shouldUpdateChunks {
		logger.Debug("%s changed view distance to %d", c.Player.Name, information.ViewDistance)
		pkt := c.NewPacket(packet.PlayClientboundSetChunkCacheRadius, proto.VarInt(information.ViewDistance))

		c.Send(pkt)
		c.updateChunkView(true)
	}
}
