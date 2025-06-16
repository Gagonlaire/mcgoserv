package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleStatusPacket(conn *Connection, pkt *packet.Packet) {
	data := mc.String(fmt.Sprintf(`{"version":{"name":"%s","protocol":%d},"players":{"max":%d,"online":%d},"description":{"text":"%s"}}`,
		mc.GameVersion, mc.ProtocolVersion, 100, 0, "Server Go Minecraft"))

	_ = pkt.ResetWith(0x0, &data)
	_ = pkt.Send(conn.Conn)
}

func HandlePingPacket(conn *Connection, pkt *packet.Packet) {
	var timestamp mc.Long

	if err := pkt.Decode(&timestamp); err != nil {
		fmt.Println("Error decoding ping packet:", err)
		return
	}

	_ = pkt.ResetWith(0x1, &timestamp)
	_ = pkt.Send(conn.Conn)
	// todo: we should gracefully close the connection, but for now it cause 'use of closed network connection' error in main loop
	// _ = conn.Conn.Close()
}
