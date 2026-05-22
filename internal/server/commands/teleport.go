package commands

import (
	"math"
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

const playerEyeHeight = 1.62

func registerTeleport(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/teleport <destination>",
			parsers.Entity.Single(true)).
			Requires(2).Executes(teleportToEntity(s))
		Build("/teleport <location>",
			parsers.Vec3).
			Requires(2).Executes(teleportToLocation(s))
		Build("/teleport <targets> <destination>",
			parsers.Entity, parsers.Entity.Single(true)).
			Requires(2).Executes(teleportToEntity(s))
		Build("/teleport <targets> <location>",
			parsers.Entity, parsers.Vec3).
			Requires(2).Executes(teleportToLocation(s))
		Build("/teleport <targets> <location> <rotation>",
			parsers.Entity, parsers.Vec3, parsers.Rotation).
			Requires(2).Executes(teleportToLocation(s))
		Build("/teleport <targets> <location> facing <facing_pos>",
			parsers.Entity, parsers.Vec3, parsers.Vec3).
			Requires(2).Executes(teleportToLocation(s))
		Build("/teleport <targets> <location> facing entity <facing_entity>",
			parsers.Entity, parsers.Vec3, parsers.Entity.Single(true)).
			Requires(2).Executes(teleportToLocation(s))
		Build("/teleport <targets> <location> facing entity <facing_entity> <anchor: eyes|feet>",
			parsers.Entity, parsers.Vec3, parsers.Entity.Single(true)).
			Requires(2).Executes(teleportToLocation(s))
	})

	s.Commander.Register(Literal("tp").Redirects(s.Commander.Resolve("teleport")))
}

func teleportTargets(s *server.Server, cc *CommandContext) []entity.Entity {
	if cc.Args.Has("targets") {
		srcUUID, srcPos := commandSource(cc)
		return s.World.ResolveTarget(cc.Args.GetEntityTarget("targets"), srcUUID, srcPos)
	}
	if e := cc.Source.Entity; e != nil {
		return []entity.Entity{e}
	}
	return nil
}

func teleportSingleEntity(s *server.Server, cc *CommandContext, name string) (entity.Entity, bool) {
	srcUUID, srcPos := commandSource(cc)
	resolved := s.World.ResolveTarget(cc.Args.GetEntityTarget(name), srcUUID, srcPos)
	if len(resolved) == 0 {
		cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
		return nil, false
	}
	return resolved[0], true
}

func teleportToEntity(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		dest, ok := teleportSingleEntity(s, cc, "destination")
		if !ok {
			return &CommandResult{Success: 0}, nil
		}
		targets := teleportTargets(s, cc)
		if len(targets) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}

		pos, rot := dest.GetPos(), dest.GetRot()
		for _, t := range targets {
			teleportEntity(s, t, pos, rot)
		}

		if len(targets) == 1 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsTeleportSuccessEntitySingle,
				entityDisplayName(targets[0]), entityDisplayName(dest)))
		} else {
			cc.SendMessage(tc.Translatable(mcdata.CommandsTeleportSuccessEntityMultiple,
				tc.Text(strconv.Itoa(len(targets))), entityDisplayName(dest)))
		}
		return &CommandResult{Success: 1, Result: len(targets)}, nil
	}
}

func teleportToLocation(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		targets := teleportTargets(s, cc)
		if len(targets) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}

		loc := GetArgument[parsers.ParsedVec3](cc.Args, "location").
			Resolve(cc.Source.Position, cc.Source.Rotation)

		var fixedRot [2]float32
		var facingPos [3]float64
		var hasFixedRot, hasFacing bool

		switch {
		case cc.Args.Has("rotation"):
			fixedRot = GetArgument[parsers.ParsedRotation](cc.Args, "rotation").Resolve(cc.Source.Rotation)
			hasFixedRot = true
		case cc.Args.Has("facing_pos"):
			facingPos = GetArgument[parsers.ParsedVec3](cc.Args, "facing_pos").
				Resolve(cc.Source.Position, cc.Source.Rotation)
			hasFacing = true
		case cc.Args.Has("facing_entity"):
			fe, ok := teleportSingleEntity(s, cc, "facing_entity")
			if !ok {
				return &CommandResult{Success: 0}, nil
			}
			facingPos = fe.GetPos()
			if cc.Args.Has("anchor") && cc.Args.GetString("anchor") == "eyes" {
				facingPos[1] += eyeHeight(fe)
			}
			hasFacing = true
		}

		for _, t := range targets {
			rot := t.GetRot()
			switch {
			case hasFixedRot:
				rot = fixedRot
			case hasFacing:
				rot = lookAt(loc, facingPos)
			}
			teleportEntity(s, t, loc, rot)
		}

		x, y, z := tc.Text(formatCoord(loc[0])), tc.Text(formatCoord(loc[1])), tc.Text(formatCoord(loc[2]))
		if len(targets) == 1 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsTeleportSuccessLocationSingle,
				entityDisplayName(targets[0]), x, y, z))
		} else {
			cc.SendMessage(tc.Translatable(mcdata.CommandsTeleportSuccessLocationMultiple,
				tc.Text(strconv.Itoa(len(targets))), x, y, z))
		}
		return &CommandResult{Success: 1, Result: len(targets)}, nil
	}
}

func teleportEntity(s *server.Server, e entity.Entity, pos [3]float64, rot [2]float32) {
	if player, ok := e.(*entity.Player); ok {
		if conn, ok := s.ConnectionsByEID.Load(player.EntityID); ok {
			conn.(*server.Connection).Teleport(pos[0], pos[1], pos[2], rot[0], rot[1], 0)
			return
		}
	}

	old := e.GetPos()
	e.SetPos(pos)
	e.SetRot(rot)
	s.World.UpdateEntityChunk(e.GetID(), old[0], old[2], pos[0], pos[2])
	if pkt, err := packet.NewPacket(packet.PlayClientboundEntityPositionSync,
		encoders.NewTeleportEntity(e.GetID(), pos, rot, e.IsOnGround())); err == nil {
		s.BroadcastEntityViewers(e, pkt)
	}
}

// lookAt returns the yaw/pitch that points from one position toward another.
func lookAt(from, to [3]float64) [2]float32 {
	dx, dy, dz := to[0]-from[0], to[1]-from[1], to[2]-from[2]
	horizontal := math.Sqrt(dx*dx + dz*dz)
	yaw := math.Atan2(-dx, dz) * 180 / math.Pi
	pitch := -math.Atan2(dy, horizontal) * 180 / math.Pi
	return [2]float32{float32(yaw), float32(pitch)}
}

func eyeHeight(e entity.Entity) float64 {
	// todo: check doc for this data
	if _, ok := e.(*entity.Player); ok {
		return playerEyeHeight
	}
	return 0
}

func formatCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
