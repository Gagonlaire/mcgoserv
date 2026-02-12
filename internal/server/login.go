package server

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/world"
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

	c.Player = world.NewPlayer(mc.VarInt(c.server.World.GetNextEntityID()), PlayerUUID, Name, c.server.World)
	_ = pkt.ResetWith(packet.LoginClientboundLoginSuccess, &PlayerUUID, &Name, &Properties)
	_ = pkt.Send(c.Conn)
}

func (c *Connection) HandleLoginAckPacket(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn)
}
