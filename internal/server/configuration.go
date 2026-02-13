package server

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/world"
	"github.com/Tnze/go-mc/nbt"
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
	c.server.World.AddPlayer(c.Player)
	c.State = mc.StatePlay
	dimensionsName := []mc.String{"minecraft:overworld", "minecraft:the_nether", "minecraft:the_end"}

	_ = pkt.ResetWith(
		packet.PlayClientboundLogin,
		mc.Int(c.Player.EntityID),
		mc.Boolean(c.server.Properties.Hardcore),
		mc.NewPrefixedArray(&dimensionsName),
		mc.VarInt(c.server.Properties.MaxPlayers),
		mc.VarInt(c.server.Properties.ViewDistance),
		mc.VarInt(c.server.Properties.SimulationDistance),
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
	_ = pkt.Send(c.Conn)

	_ = pkt.ResetWith(
		packet.PlayClientboundSynchronizePlayerPosition,
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
	_ = pkt.Send(c.Conn)

	_ = pkt.ResetWith(
		packet.PlayClientboundGameEvent,
		mc.UnsignedByte(13),
		mc.Float(0.0),
	)
	_ = pkt.Send(c.Conn)

	players := []*world.Player{c.Player}
	allPlayers := slices.Collect(maps.Values(c.server.World.Players))

	actions := mc.ActionAddPlayer | mc.ActionUpdateListed
	pkt1, _ := packet.BuildPlayerInfoUpdatePacket(actions, players)
	c.server.Broadcaster.Broadcast(pkt1, systems.NotSender(c))
	pkt1, _ = packet.BuildPlayerInfoUpdatePacket(actions, allPlayers)
	_ = pkt1.Send(c.Conn)

	_ = pkt.ResetWith(
		packet.PlayClientboundSetTime,
		mc.Long(c.server.World.Time),
		mc.Long(c.server.World.DayTime),
		mc.Boolean(true),
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
	uuid := mc.UUID(c.Player.UUID)
	// spawn newly connected player
	pkt, _ = packet.NewPacket(
		packet.PlayClientboundSpawnEntity,
		&eID2,
		&uuid,
		&entityType,
		mc.Double(c.Player.Pos[0]),
		mc.Double(c.Player.Pos[1]),
		mc.Double(c.Player.Pos[2]),
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
			uuid := mc.UUID(conn.Player.UUID)

			pkt, _ := packet.NewPacket(
				packet.PlayClientboundSpawnEntity,
				mc.VarInt(conn.Player.EntityID),
				&uuid,
				entityType,
				mc.Double(conn.Player.Pos[0]),
				mc.Double(conn.Player.Pos[1]),
				mc.Double(conn.Player.Pos[2]),
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

	// todo: replace with a flexible type
	type PlayerJoined struct {
		Text  string `nbt:"text"`
		Color string `nbt:"color,omitempty"`
	}
	component := PlayerJoined{
		Text:  string(c.Player.Name) + " joined the game",
		Color: "yellow",
	}
	pkt, _ = packet.NewPacket(packet.PlayClientboundSystemChat)
	encoder := nbt.NewEncoder(pkt.Buffer)
	encoder.NetworkFormat(true)
	_ = encoder.Encode(component, "")
	_ = pkt.Encode(mc.Boolean(false))
	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}
