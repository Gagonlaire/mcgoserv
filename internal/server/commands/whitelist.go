package commands

import (
	"fmt"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal"
	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func enforceWhitelist(s *server.Server) {
	s.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*server.Connection)
		if conn.Player == nil {
			return true
		}
		if isOp, _ := s.PlayerRegistry.IsOp(uuid.UUID(conn.Player.UUID)); isOp {
			return true
		}
		if !s.PlayerRegistry.IsWhitelisted(uuid.UUID(conn.Player.UUID)) {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
		}
		return true
	})
}

func kickRemovedPlayer(s *server.Server, uuidStr string) {
	removedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return
	}
	if isOp, _ := s.PlayerRegistry.IsOp(removedUUID); isOp {
		return
	}
	s.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*server.Connection)
		if conn.Player != nil && conn.Player.UUID == entities.NbtUUID(removedUUID) {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
			return false
		}
		return true
	})
}

func registerWhitelist(s *server.Server) {
	s.Commander.Register(
		Literal("whitelist").Requires(3).Connect(
			Literal("add").Connect(
				Argument("targets", parsers.GameProfile).
					Executes(func(cc *CommandContext) (*CommandResult, error) {
						target := cc.Args.GetEntityTarget("targets")

						type whitelistTarget struct {
							UUID uuid.UUID
							Name string
						}
						var targets []whitelistTarget

						switch target.Type {
						case mc.TargetTypeUUID:
							if s.Config.Security.OnlineMode {
								name, err := api.GetProfileNameByUUID(target.UUID)
								if err != nil {
									cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
									return &CommandResult{Success: 0, Result: 0}, nil
								}
								targets = append(targets, whitelistTarget{target.UUID, name})
							} else {
								targets = append(targets, whitelistTarget{target.UUID, "Unknown"})
							}
						case mc.TargetTypePlayerName:
							if s.Config.Security.OnlineMode {
								u, realName, err := api.GetUserUUID(target.Name)
								if err != nil {
									cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
									return &CommandResult{Success: 0, Result: 0}, nil
								}
								targets = append(targets, whitelistTarget{u, realName})
							} else {
								offlineUUID := internal.GetOfflineUUID(target.Name)
								targets = append(targets, whitelistTarget{offlineUUID, target.Name})
							}
						case mc.TargetTypeSelector:
							var sourceUUID uuid.UUID
							var sourcePos [3]float64
							if player, ok := cc.Source.Entity.(*entities.Player); ok {
								sourceUUID = uuid.UUID(player.UUID)
								sourcePos = player.Position
							}
							resolved := s.World.ResolveTarget(target, sourceUUID, sourcePos)
							for _, p := range resolved {
								targets = append(targets, whitelistTarget{uuid.UUID(p.UUID), p.Name})
							}
							if len(resolved) == 0 {
								return &CommandResult{Success: 0, Result: 0}, nil
							}
						}

						success := 0
						for _, t := range targets {
							if s.PlayerRegistry.IsWhitelisted(t.UUID) {
								cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAddFailed))
								continue
							}
							s.PlayerRegistry.AddWhitelist(t.UUID, t.Name)
							cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAddSuccess, tc.Text(t.Name)))
							success++
						}

						if success > 0 && s.Config.Security.Whitelist.Enforce {
							enforceWhitelist(s)
						}

						return &CommandResult{Success: success, Result: 0}, nil
					}),
			),
			Literal("remove").Connect(
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
							entry, ok := s.PlayerRegistry.RemoveWhitelistByUUID(target.UUID.String())
							if ok {
								removals = append(removals, removedInfo{entry.UUID, entry.Name})
							}
						case mc.TargetTypePlayerName:
							caseSensitive := !s.Config.Security.OnlineMode
							entry, ok := s.PlayerRegistry.RemoveWhitelistByName(target.Name, caseSensitive)
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
							resolved := s.World.ResolveTarget(target, sourceUUID, sourcePos)
							for _, p := range resolved {
								entry, ok := s.PlayerRegistry.RemoveWhitelistByUUID(uuid.UUID(p.UUID).String())
								if ok {
									removals = append(removals, removedInfo{entry.UUID, entry.Name})
								}
							}
						}

						if len(removals) == 0 {
							cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistRemoveFailed))
							return &CommandResult{Success: 0, Result: 0}, nil
						}

						for _, r := range removals {
							cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistRemoveSuccess, tc.Text(r.Name)))
							if s.Config.Security.Whitelist.Enforce {
								kickRemovedPlayer(s, r.UUID)
							}
						}

						return &CommandResult{Success: len(removals), Result: 0}, nil
					}),
			),
			Literal("list").Executes(func(cc *CommandContext) (*CommandResult, error) {
				s.PlayerRegistry.Mu.RLock()
				whitelist := make([]string, len(s.PlayerRegistry.Whitelist))
				for i, entry := range s.PlayerRegistry.Whitelist {
					whitelist[i] = entry.Name
				}
				s.PlayerRegistry.Mu.RUnlock()

				if len(whitelist) == 0 {
					cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistNone))
				} else {
					cc.SendMessage(tc.Translatable(
						mcdata.CommandsWhitelistList,
						tc.Text(fmt.Sprintf("%d", len(whitelist))),
						tc.Text(strings.Join(whitelist, ", ")),
					))
				}

				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("on").Executes(func(cc *CommandContext) (*CommandResult, error) {
				if s.Config.Security.Whitelist.Enabled {
					cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAlreadyOn))
					return &CommandResult{Success: 0, Result: 0}, nil
				}

				s.Config.Security.Whitelist.Enabled = true
				_ = systems.SaveConfig("config.yml", s.Config)

				cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistEnabled))

				if s.Config.Security.Whitelist.Enforce {
					enforceWhitelist(s)
				}

				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("off").Executes(func(cc *CommandContext) (*CommandResult, error) {
				if !s.Config.Security.Whitelist.Enabled {
					cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAlreadyOff))
					return &CommandResult{Success: 0, Result: 0}, nil
				}

				s.Config.Security.Whitelist.Enabled = false
				_ = systems.SaveConfig("config.yml", s.Config)

				cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistDisabled))

				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("reload").Executes(func(cc *CommandContext) (*CommandResult, error) {
				s.PlayerRegistry.ReloadWhitelist()
				cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistReloaded))

				if s.Config.Security.Whitelist.Enforce {
					enforceWhitelist(s)
				}

				return &CommandResult{Success: 1, Result: 0}, nil
			}),
		),
	)
}
