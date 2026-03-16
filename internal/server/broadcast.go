package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Filter func(target *Connection) bool

// BroadcastAll sends a packet to every player. Takes ownership of the packet.
func (s *Server) BroadcastAll(pkt *packet.OutboundPacket, filters ...Filter) {
	s.iteratePlay(func(conn *Connection) {
		sendFiltered(conn, pkt, filters)
	})
	pkt.Free()
}

// BroadcastOthers sends a packet to every player except sender. Takes ownership of the packet.
func (s *Server) BroadcastOthers(sender *Connection, pkt *packet.OutboundPacket, filters ...Filter) {
	s.iteratePlay(func(conn *Connection) {
		if conn != sender {
			sendFiltered(conn, pkt, filters)
		}
	})
	pkt.Free()
}

// BroadcastViewers sends a packet to players watching the sender's chunk, excluding the sender. Takes ownership of the packet.
func (s *Server) BroadcastViewers(sender *Connection, pkt *packet.OutboundPacket, filters ...Filter) {
	dim := world.GetEntityDimension(&sender.Player.LivingEntity.BaseEntity)
	cx, cz := world.GetChunkPosition(sender.Player.Pos[0], sender.Player.Pos[2])
	chunk := dim.GetChunk(cx, cz)

	senderEID := sender.Player.EntityID
	for watcherID := range chunk.Watchers {
		if watcherID == senderEID {
			continue
		}
		if conn, ok := s.ConnectionsByEID[watcherID]; ok {
			sendFiltered(conn, pkt, filters)
		}
	}
	pkt.Free()
}

func sendFiltered(conn *Connection, pkt *packet.OutboundPacket, filters []Filter) {
	for _, f := range filters {
		if !f(conn) {
			return
		}
	}
	pkt.Retain()
	select {
	case conn.OutboundPackets <- pkt:
	case <-conn.ctx.Done():
		pkt.Free()
	}
}

func (s *Server) iteratePlay(fn func(*Connection)) {
	s.Connections.Range(func(key, _ any) bool {
		conn := key.(*Connection)
		// todo: it should also check if the player is loaded but this ignore some packets (like the chat init)
		if conn.State == mc.StatePlay {
			fn(conn)
		}
		return true
	})
}
