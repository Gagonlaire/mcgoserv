package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
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

	_ = pkt.ResetWith(packet.LoginClientboundLoginSuccess, &PlayerUUID, &Name, &Properties)

	if err := pkt.Send(c.Conn); err != nil {
		fmt.Println("Error sending loginStart packet:", err)
		return
	}
}

func (c *Connection) HandleLoginAckPacket(pkt *packet.Packet) {
	c.State = mc.StateConfiguration

	_ = pkt.ResetWith(packet.ConfigurationClientboundKnownPacks, &mc.ServerDataPacks)
	if err := pkt.Send(c.Conn); err != nil {
		fmt.Println("Error sending loginAck packet:", err)
		return
	}
}
