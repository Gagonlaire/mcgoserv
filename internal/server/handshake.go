package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type HandshakePacket struct {
	ProtocolVersion int32  `mc:"varint"`
	ServerAddress   string `mc:"string"`
	ServerPort      uint16 `mc:"u16"`
	Intent          int32  `mc:"varint"`
}

func HandleHandshakePacket(conn *Connection, pkt *packet.Packet) {
	var handshake HandshakePacket

	pkt.Decode(&handshake)
	if handshake.Intent == 1 || handshake.Intent == 2 {
		conn.State = ConnState(handshake.Intent)
	}
}
