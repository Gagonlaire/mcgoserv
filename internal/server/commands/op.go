package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerOp(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/op <targets>", parsers.GameProfile).Requires(3).Executes(opAdd(s))
	})
}

func registerDeop(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/deop <targets>", parsers.GameProfile).Requires(3).Executes(opRemove(s))
	})
}

func opAdd(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetEntityTarget("targets")
		targets := resolveProfileTargets(s, cc, target)
		if len(targets) == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		opLevel := s.Config.Security.OpLevel
		affected := 0
		for _, t := range targets {
			if isOp, _ := s.PlayerRegistry.IsOp(t.UUID); isOp {
				cc.SendMessage(tc.Translatable(mcdata.CommandsOpFailed))
				continue
			}
			s.PlayerRegistry.AddOp(t.UUID, t.Name, opLevel, false)
			cc.SendMessage(tc.Translatable(mcdata.CommandsOpSuccess, tc.Text(t.Name)))

			if p := s.World.PlayersByUUID[t.UUID]; p != nil {
				p.PermissionLevel = opLevel
				if conn, ok := s.ConnectionsByEID.Load(p.EntityID); ok {
					_ = s.SendCommands(conn.(*server.Connection))
				}
			}
			affected++
		}

		if affected == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}
		return &CommandResult{Success: 1, Result: affected}, nil
	}
}

func opRemove(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetEntityTarget("targets")

		type removedInfo struct {
			UUID string
			Name string
		}
		var removals []removedInfo

		switch target.Type {
		case mc.TargetTypeUUID:
			if entry, ok := s.PlayerRegistry.RemoveOpByUUID(target.UUID.String()); ok {
				removals = append(removals, removedInfo{entry.UUID, entry.Name})
			}
		case mc.TargetTypePlayerName:
			caseSensitive := !s.Config.Security.OnlineMode
			if entry, ok := s.PlayerRegistry.RemoveOpByName(target.Name, caseSensitive); ok {
				removals = append(removals, removedInfo{entry.UUID, entry.Name})
			}
		case mc.TargetTypeSelector:
			sourceUUID, sourcePos := commandSource(cc)
			resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
			for _, p := range resolved {
				if entry, ok := s.PlayerRegistry.RemoveOpByUUID(uuid.UUID(p.UUID).String()); ok {
					removals = append(removals, removedInfo{entry.UUID, entry.Name})
				}
			}
		}

		if len(removals) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDeopFailed))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		for _, r := range removals {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDeopSuccess, tc.Text(r.Name)))

			removedUUID, err := uuid.Parse(r.UUID)
			if err != nil {
				continue
			}
			if p := s.World.PlayersByUUID[removedUUID]; p != nil {
				p.PermissionLevel = 0
				if conn, ok := s.ConnectionsByEID.Load(p.EntityID); ok {
					_ = s.SendCommands(conn.(*server.Connection))
				}
			}
		}

		return &CommandResult{Success: 1, Result: len(removals)}, nil
	}
}
