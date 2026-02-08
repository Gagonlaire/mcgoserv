package server

import (
	"log"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func (c *Connection) HandleHandshakePacket(pkt *packet.Packet) {
	var (
		ProtocolVersion mc.VarInt
		ServerAddress   mc.String
		ServerPort      mc.UnsignedShort
		Intent          mc.VarInt
	)

	if err := pkt.Decode(&ProtocolVersion, &ServerAddress, &ServerPort, &Intent); err != nil {
		log.Println("Error decoding handshake packet:", err)
		return
	}

	if Intent == mc.VarInt(mc.StateStatus) || Intent == mc.VarInt(mc.StateLogin) {
		c.State = mc.State(Intent)
	}
}
