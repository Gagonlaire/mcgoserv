package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
)

func (s *Server) SpawnEntity(entity entities.Entity) error {
	if err := s.World.AddEntity(entity); err != nil {
		return err
	}
	s.broadcastSpawn(entity)
	return nil
}

func (s *Server) DespawnEntity(entity entities.Entity) {
	s.broadcastDespawn(entity)
	s.World.RemoveEntity(entity)
}

func (s *Server) SpawnPlayer(player *entities.Player, dimensionID world.DimensionID) error {
	if err := s.World.AddPlayer(player, dimensionID); err != nil {
		return err
	}
	s.broadcastSpawn(player)
	return nil
}

func (s *Server) DespawnPlayer(player *entities.Player) {
	s.broadcastDespawn(player)
	s.World.RemovePlayer(player)
}

func (s *Server) broadcastSpawn(entity entities.Entity) {
	pkt, err := packet.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(entity))
	if err != nil {
		return
	}
	s.BroadcastEntityViewers(entity, pkt)
}

func (s *Server) broadcastDespawn(entity entities.Entity) {
	pkt, err := packet.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), mc.VarInt(entity.GetID()))
	if err != nil {
		return
	}
	s.BroadcastEntityViewers(entity, pkt)
}
