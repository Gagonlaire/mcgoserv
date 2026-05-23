package commands

import (
	"fmt"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
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
		if conn.Player != nil && conn.Player.UUID == entity.NbtUUID(removedUUID) {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
			return false
		}
		return true
	})
}

func registerWhitelist(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/whitelist add <targets>", parsers.GameProfile).Requires(3).Executes(whitelistAdd(s))
		Build("/whitelist remove <targets>", parsers.GameProfile).Requires(3).Executes(whitelistRemove(s))
		Build("/whitelist list").Requires(3).Executes(whitelistList(s))
		Build("/whitelist on").Requires(3).Executes(whitelistOn(s))
		Build("/whitelist off").Requires(3).Executes(whitelistOff(s))
		Build("/whitelist reload").Requires(3).Executes(whitelistReload(s))
	})
}

func whitelistAdd(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetEntityTarget("targets")
		targets := resolveProfileTargets(s, cc, target)
		if len(targets) == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		affected := 0
		for _, t := range targets {
			if s.PlayerRegistry.IsWhitelisted(t.UUID) {
				cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAddFailed))
				continue
			}
			s.PlayerRegistry.AddWhitelist(t.UUID, t.Name)
			cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAddSuccess, tc.Text(t.Name)))
			affected++
		}

		if affected > 0 && s.Config.Security.Whitelist.Enforce {
			enforceWhitelist(s)
		}

		if affected == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}
		return &CommandResult{Success: 1, Result: affected}, nil
	}
}

func whitelistRemove(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
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
			sourceUUID, sourcePos := commandSource(cc)
			resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
			for _, p := range resolved {
				if entry, ok := s.PlayerRegistry.RemoveWhitelistByUUID(uuid.UUID(p.UUID).String()); ok {
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

		return &CommandResult{Success: 1, Result: len(removals)}, nil
	}
}

func whitelistList(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
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
	}
}

func whitelistOn(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
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
	}
}

func whitelistOff(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		if !s.Config.Security.Whitelist.Enabled {
			cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistAlreadyOff))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		s.Config.Security.Whitelist.Enabled = false
		_ = systems.SaveConfig("config.yml", s.Config)

		cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistDisabled))

		return &CommandResult{Success: 1, Result: 0}, nil
	}
}

func whitelistReload(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		s.PlayerRegistry.ReloadWhitelist()
		cc.SendMessage(tc.Translatable(mcdata.CommandsWhitelistReloaded))

		if s.Config.Security.Whitelist.Enforce {
			enforceWhitelist(s)
		}

		return &CommandResult{Success: 1, Result: 0}, nil
	}
}
