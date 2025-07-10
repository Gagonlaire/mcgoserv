package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"time"
)

func (c *Connection) HandleConfirmTeleportationPacket(pkt *packet.Packet) {
	var teleportId mc.VarInt

	if err := pkt.Decode(&teleportId); err != nil {
		return
	}
}

func (c *Connection) HandleKeepAlivePacket(pkt *packet.Packet) {
	var keepAliveId mc.Long

	if err := pkt.Decode(&keepAliveId); err != nil {
		return
	}

	c.LastKeepAliveID = int64(keepAliveId)
	c.LastKeepAlive = time.Now()
}
