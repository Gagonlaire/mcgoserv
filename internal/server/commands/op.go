package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal"
	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerOp(s *server.Server) {
	s.Commander.Register(
		Literal("op").Requires(3).Connect(
			Argument("targets", parsers.GameProfile).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					target := cc.Args.GetEntityTarget("targets")

					type opTarget struct {
						UUID uuid.UUID
						Name string
					}
					var targets []opTarget

					switch target.Type {
					case mc.TargetTypeUUID:
						if s.Config.Security.OnlineMode {
							name, err := api.GetProfileNameByUUID(target.UUID)
							if err != nil {
								cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
								return &CommandResult{Success: 0, Result: 0}, nil
							}
							targets = append(targets, opTarget{target.UUID, name})
						} else {
							targets = append(targets, opTarget{target.UUID, "Unknown"})
						}
					case mc.TargetTypePlayerName:
						if s.Config.Security.OnlineMode {
							u, realName, err := api.GetUserUUID(target.Name)
							if err != nil {
								cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
								return &CommandResult{Success: 0, Result: 0}, nil
							}
							targets = append(targets, opTarget{u, realName})
						} else {
							offlineUUID := internal.GetOfflineUUID(target.Name)
							targets = append(targets, opTarget{offlineUUID, target.Name})
						}
					case mc.TargetTypeSelector:
						var sourceUUID uuid.UUID
						var sourcePos [3]float64
						if player, ok := cc.Source.Entity.(*entities.Player); ok {
							sourceUUID = uuid.UUID(player.UUID)
							sourcePos = player.Position
						}
						resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
						for _, p := range resolved {
							targets = append(targets, opTarget{uuid.UUID(p.UUID), p.Name})
						}
						if len(resolved) == 0 {
							return &CommandResult{Success: 0, Result: 0}, nil
						}
					}

					opLevel := s.Config.Security.OpLevel
					success := 0
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
						success++
					}

					return &CommandResult{Success: success, Result: 0}, nil
				}),
		),
	)
}

func registerDeop(s *server.Server) {
	s.Commander.Register(
		Literal("deop").Requires(3).Connect(
			Argument("targets", parsers.GameProfile).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					target := cc.Args.GetEntityTarget("targets")

					type removedInfo struct {
						UUID string
						Name string
					}
					var removals []removedInfo

					switch target.Type {
					case mc.TargetTypeUUID:
						entry, ok := s.PlayerRegistry.RemoveOpByUUID(target.UUID.String())
						if ok {
							removals = append(removals, removedInfo{entry.UUID, entry.Name})
						}
					case mc.TargetTypePlayerName:
						caseSensitive := !s.Config.Security.OnlineMode
						entry, ok := s.PlayerRegistry.RemoveOpByName(target.Name, caseSensitive)
						if ok {
							removals = append(removals, removedInfo{entry.UUID, entry.Name})
						}
					case mc.TargetTypeSelector:
						var sourceUUID uuid.UUID
						var sourcePos [3]float64
						if player, ok := cc.Source.Entity.(*entities.Player); ok {
							sourceUUID = uuid.UUID(player.UUID)
							sourcePos = player.Position
						}
						resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
						for _, p := range resolved {
							entry, ok := s.PlayerRegistry.RemoveOpByUUID(uuid.UUID(p.UUID).String())
							if ok {
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

					return &CommandResult{Success: len(removals), Result: 0}, nil
				}),
		),
	)
}
