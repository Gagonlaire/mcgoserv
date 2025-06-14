package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleHandshakePacket(conn *Connection, pkt *packet.Packet) {
	var (
		ProtocolVersion mc.VarInt
		ServerAddress   mc.String
		ServerPort      mc.UnsignedShort
		Intent          mc.VarInt
	)

	if err := pkt.Decode(&ProtocolVersion, &ServerAddress, &ServerPort, &Intent); err != nil {
		fmt.Println("Error decoding handshake packet:", err)
		return
	}

	if Intent == mc.VarInt(StateStatus) || Intent == mc.VarInt(StateLogin) {
		conn.State = State(Intent)
	}
}
