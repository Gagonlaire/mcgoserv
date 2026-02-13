package server

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/world"
	"github.com/google/uuid"
)

func (c *Connection) HandleLoginStartPacket(pkt *packet.Packet) {
	var (
		Name       mc.String
		PlayerUUID mc.UUID
		// Properties todo: should fetch user data from mc api (array of properties for capes, skins, etc.)
		Properties = mc.VarInt(0)
	)

	if err := pkt.Decode(&Name, &PlayerUUID); err != nil {
		fmt.Println("Error decoding loginStart packet:", err)
		return
	}

	c.server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player != nil && conn.Player.UUID == uuid.UUID(PlayerUUID) {
			conn.Disconnect("You have logged in from another location.")
			return false
		}
		return true
	})

	c.Player = world.NewPlayer(uuid.UUID(PlayerUUID), Name, c.server.World, c.server.Properties)
	_ = pkt.ResetWith(packet.LoginClientboundLoginSuccess, &PlayerUUID, &Name, &Properties)
	_ = pkt.Send(c.Conn)
}

func (c *Connection) HandleLoginAckPacket(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn)
}
