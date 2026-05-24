package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func (s *Server) SpawnEntity(entity entity.Entity) error {
	if err := s.World.AddEntity(entity); err != nil {
		return err
	}
	s.broadcastSpawn(entity)
	return nil
}

func (s *Server) DespawnEntity(entity entity.Entity) {
	s.broadcastDespawn(entity)
	s.World.RemoveEntity(entity)
}

func (s *Server) SpawnPlayer(player *entity.Player, dimensionID world.DimensionID) error {
	if err := s.World.AddPlayer(player, dimensionID); err != nil {
		return err
	}
	s.broadcastSpawn(player)
	return nil
}

func (s *Server) DespawnPlayer(player *entity.Player) {
	s.broadcastDespawn(player)
	s.World.RemovePlayer(player)
}

func (s *Server) broadcastSpawn(entity entity.Entity) {
	pkt, err := packet.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(entity))
	if err != nil {
		return
	}
	s.BroadcastEntityViewers(entity, pkt)
}

func (s *Server) broadcastDespawn(entity entity.Entity) {
	pkt, err := packet.NewPacket(packet.PlayClientboundRemoveEntities, proto.VarInt(1), proto.VarInt(entity.GetID()))
	if err != nil {
		return
	}
	s.BroadcastEntityViewers(entity, pkt)
}
