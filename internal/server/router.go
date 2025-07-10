package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"log"
)

type PacketHandler func(c *Connection, pkt *packet.Packet)

type PacketRouter struct {
	handlers []map[int]PacketHandler
}

func NewPacketRouter() *PacketRouter {
	r := &PacketRouter{
		handlers: make([]map[int]PacketHandler, mc.StateMax),
	}
	for i := range r.handlers {
		r.handlers[i] = make(map[int]PacketHandler)
	}
	r.registerHandlers()
	return r
}

func (r *PacketRouter) Handle(conn *Connection, pkt *packet.Packet) {
	stateHandlers := r.handlers[conn.State]
	if handler, ok := stateHandlers[int(pkt.ID)]; ok {
		handler(conn, pkt)
	} else {
		log.Printf("unhandled packet ID 0x%X in state %d", pkt.ID, conn.State)
	}
}

func (r *PacketRouter) registerHandlers() {
	// State Handshake
	r.handlers[mc.StateHandshake][packet.HandshakeServerboundHandshake] = (*Connection).HandleHandshakePacket

	// State Status
	r.handlers[mc.StateStatus][packet.StatusServerboundStatusRequest] = (*Connection).HandleStatusRequestPacket
	r.handlers[mc.StateStatus][packet.StatusServerboundPing] = (*Connection).HandlePingPacket

	// State Login
	r.handlers[mc.StateLogin][packet.LoginServerboundLoginStart] = (*Connection).HandleLoginStartPacket
	r.handlers[mc.StateLogin][packet.LoginServerboundLoginAcknowledged] = (*Connection).HandleLoginAckPacket

	// State Configuration
	r.handlers[mc.StateConfiguration][packet.ConfigurationServerboundAcknowledgeFinishConfiguration] = (*Connection).HandleFinishConfigurationAckPacket
	r.handlers[mc.StateConfiguration][packet.ConfigurationServerboundKeepAlive] = (*Connection).HandleKeepAlivePacket
	r.handlers[mc.StateConfiguration][packet.ConfigurationServerboundKnownPacks] = (*Connection).HandleClientKnownPacksPacket

	// State Play
	r.handlers[mc.StatePlay][packet.PlayServerboundConfirmTeleportation] = (*Connection).HandleConfirmTeleportationPacket
	r.handlers[mc.StatePlay][packet.PlayServerboundKeepAlive] = (*Connection).HandleKeepAlivePacket
}
