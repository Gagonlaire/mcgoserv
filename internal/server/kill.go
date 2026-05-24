package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
)

const deathAnimationTicks = 20

// Kill triggers the death flow for any entity. Non-living entities are despawned immediately.
func (s *Server) Kill(target entity.Entity, killer entity.Entity, cause tc.Component) {
	if target == nil {
		return
	}

	if player, ok := target.(*entity.Player); ok {
		s.killPlayer(player, killer, cause)
		return
	}

	if living := asLiving(target); living != nil {
		s.killMob(target, living)
		return
	}

	s.DespawnEntity(target)
}

func (s *Server) killPlayer(player *entity.Player, killer entity.Entity, cause tc.Component) {
	if player.IsDying {
		return
	}
	player.IsDying = true
	player.Health = 0
	player.SetPose(entity.PoseDying)
	s.World.EnqueueDirty(player)

	conn, ok := s.ConnectionsByEID.Load(player.EntityID)
	if !ok {
		return
	}
	c := conn.(*Connection)

	healthPkt := c.NewPacket(
		packet.PlayClientboundSetHealth,
		proto.Float(0),
		proto.VarInt(player.FoodLevel),
		proto.Float(player.FoodSaturationLevel),
	)
	c.Send(healthPkt)

	combatPkt := c.NewPacket(
		packet.PlayClientboundPlayerCombatKill,
		proto.VarInt(player.EntityID),
		cause,
	)
	c.Send(combatPkt)

	s.broadcastEntityStatus(player, 3)
}

func (s *Server) killMob(target entity.Entity, living *entity.LivingEntity) {
	if living.IsDying {
		return
	}
	living.IsDying = true
	living.Health = 0
	target.Base().SetPose(entity.PoseDying)
	s.World.EnqueueDirty(target)

	s.broadcastEntityStatus(target, 3)
	s.World.DyingEntities = append(s.World.DyingEntities, target)
}

func (s *Server) Respawn(player *entity.Player) {
	if !player.IsDying {
		return
	}
	conn, ok := s.ConnectionsByEID.Load(player.EntityID)
	if !ok {
		return
	}
	c := conn.(*Connection)

	player.Motion = [3]float64{0, 0, 0}
	player.Health = 20
	player.IsDying = false
	player.DeathTime = 0
	player.SetPose(entity.PoseStanding)
	s.World.EnqueueDirty(player)

	respawnPkt := c.NewPacket(packet.PlayClientboundRespawn, &encoders.Respawn{
		DimensionType:    0,
		DimensionName:    "overworld",
		HashedSeed:       1,
		GameMode:         proto.UnsignedByte(player.GameMode),
		PreviousGameMode: proto.Byte(player.PreviousGameMode),
		IsDebug:          false,
		IsFlat:           false,
		HasDeathLocation: false,
		PortalCooldown:   100,
		SeaLevel:         64,
		DataKept:         0,
	})
	_ = c.SendSync(respawnPkt)

	healthPkt := c.NewPacket(
		packet.PlayClientboundSetHealth,
		proto.Float(20),
		proto.VarInt(player.FoodLevel),
		proto.Float(player.FoodSaturationLevel),
	)
	_ = c.SendSync(healthPkt)

	chunkWaitPkt := c.NewPacket(
		packet.PlayClientboundGameEvent,
		proto.UnsignedByte(13),
		proto.Float(0.0),
	)
	_ = c.SendSync(chunkWaitPkt)

	spawn := s.World.SpawnPos
	c.Teleport(spawn[0], spawn[1], spawn[2], 0, 0, 0)

	logger.Debug("%s respawned at (%f, %f, %f)", logger.Identity(player.Name), spawn[0], spawn[1], spawn[2])
}

func (s *Server) broadcastEntityStatus(e entity.Entity, status byte) {
	pkt, err := packet.NewPacket(
		packet.PlayClientboundEntityEvent,
		proto.Int(e.GetID()),
		proto.Byte(status),
	)
	if err != nil {
		return
	}
	s.BroadcastEntityViewers(e, pkt)
}

func asLiving(e entity.Entity) *entity.LivingEntity {
	type livingHolder interface {
		LivingBase() *entity.LivingEntity
	}
	if h, ok := e.(livingHolder); ok {
		return h.LivingBase()
	}
	return nil
}
