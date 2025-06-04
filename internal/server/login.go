package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcproto"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/google/uuid"
)

type LoginStartPacket struct {
	Name       string    `mc:"string"`
	PlayerUUID uuid.UUID `mc:"uuid"`
}

func HandleLoginStartPacket(conn *Connection, pkt *packet.Packet) {
	var loginStart LoginStartPacket

	pkt.Decode(&loginStart)
	pkt.Buffer.Reset()
	_ = mcproto.WriteVarInt(pkt.Buffer, 0x2)
	_ = mcproto.WriteUUID(pkt.Buffer, loginStart.PlayerUUID)
	_ = mcproto.WriteString(pkt.Buffer, loginStart.Name)
	// todo: replace with a actual array func
	// this is supposed to handle player skin/cape data
	_ = mcproto.WriteVarInt(pkt.Buffer, 0)
	_ = pkt.Send(conn.Conn)
}

func HandleLoginAckPacket(conn *Connection) {
	conn.State = Configuration
}
