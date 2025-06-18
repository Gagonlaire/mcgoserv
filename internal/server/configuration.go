package server

import (
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func HandleClientKnownPacksPacket(conn *Connection, pkt *packet.Packet) {
	// todo: find why i need to call NewPArray and can't just set a var instead
	// example: var knownPacks mc.PArray[mc.DataPackIdentifier]
	var knownPacks []mc.DataPackIdentifier

	if err := pkt.Decode(mc.NewPArray(&knownPacks)); err != nil {
		fmt.Println("Error decoding clientKnownPacks packet:", err)
		return
	}

	// Server should compute the difference to know if it needs to send packs data
	// For now, we ignore this information
	for _, registryData := range mc.RegistriesData {
		_ = pkt.ResetWith(0x07, &registryData)
		_ = pkt.Send(conn.Conn)
	}

	_ = pkt.ResetWith(0x03)
	_ = pkt.Send(conn.Conn)
}

func HandleFinishConfigurationAckPacket(conn *Connection, _ *packet.Packet) {
	conn.State = StatePlay
	fmt.Println("Received finishConfigurationAckPacket")
}
