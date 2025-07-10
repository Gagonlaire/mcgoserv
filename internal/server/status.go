package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func (c *Connection) HandleStatusRequestPacket(pkt *packet.Packet) {
	data := mc.String(fmt.Sprintf(`{"version":{"name":"%s","protocol":%d},"players":{"max":%d,"online":%d},"description":{"text":"%s"}}`,
		mc.GameVersion, mc.ProtocolVersion, 100, 0, "Server Go Minecraft"))

	_ = pkt.ResetWith(packet.StatusClientboundStatusResponse, &data)
	_ = pkt.Send(c.Conn)
}

func (c *Connection) HandlePingPacket(pkt *packet.Packet) {
	var timestamp mc.Long

	if err := pkt.Decode(&timestamp); err != nil {
		fmt.Println("Error decoding ping packet:", err)
		return
	}

	_ = pkt.ResetWith(packet.StatusClientboundPongResponse, &timestamp)
	_ = pkt.Send(c.Conn)
	c.close()
}
