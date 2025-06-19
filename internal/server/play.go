package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleConfirmTeleportationPacket(_ *Connection, pkt *packet.Packet) {
	var teleportId mc.VarInt

	if err := pkt.Decode(&teleportId); err != nil {
		fmt.Println("HandleConfirmTeleportationPacket:", err)
		return
	}
	fmt.Println("HandleConfirmTeleportationPacket:", teleportId)
}
