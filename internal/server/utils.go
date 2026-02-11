package server

import "github.com/Gagonlaire/mcgoserv/internal/packet"

func (c *Connection) Send(pkt *packet.Packet) {
	select {
	case c.OutboundPackets <- pkt:
	default:
		pkt.Free()
	}
}
