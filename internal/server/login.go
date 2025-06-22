package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleLoginStartPacket(conn *Connection, pkt *packet.Packet) {
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

	_ = pkt.ResetWith(0x2, &PlayerUUID, &Name, &Properties)

	if err := pkt.Send(conn.Conn); err != nil {
		fmt.Println("Error sending loginStart packet:", err)
		return
	}
}

func HandleLoginAckPacket(conn *Connection, pkt *packet.Packet) {
	conn.State = StateConfiguration

	_ = pkt.ResetWith(0x0E, &mc.ServerDataPacks)
	if err := pkt.Send(conn.Conn); err != nil {
		fmt.Println("Error sending loginAck packet:", err)
		return
	}
}
