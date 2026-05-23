package commands

import (
	"net"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func kickBannedPlayer(s *server.Server, playerUUID entity.NbtUUID, reason string) {
	s.Connections.Range(func(k, v any) bool {
		conn := k.(*server.Connection)
		if conn.Player != nil && conn.Player.UUID == playerUUID {
			if reason != "" {
				conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedReason, tc.Text(reason)))
			} else {
				conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBanned))
			}
			return false
		}
		return true
	})
}

func kickBannedIP(s *server.Server, ip string, reason string) {
	s.Connections.Range(func(k, v any) bool {
		conn := k.(*server.Connection)
		host, _, err := net.SplitHostPort(conn.Conn.RemoteAddr().String())
		if err != nil {
			return true
		}
		if host == ip {
			if reason != "" {
				conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedIpReason, tc.Text(reason)))
			} else {
				conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedIpReason,
					tc.Translatable(mcdata.MultiplayerDisconnectBannedReasonDefault)))
			}
		}
		return true
	})
}

func resolveIPTarget(s *server.Server, target string) (string, bool) {
	if ip := net.ParseIP(target); ip != nil {
		return ip.String(), true
	}

	var found string
	s.Connections.Range(func(k, v any) bool {
		conn := k.(*server.Connection)
		if conn.Player != nil && conn.Player.Name == target {
			host, _, err := net.SplitHostPort(conn.Conn.RemoteAddr().String())
			if err == nil {
				found = host
			}
			return false
		}
		return true
	})
	if found != "" {
		return found, true
	}
	return "", false
}

func registerBan(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/ban <targets> [<reason>]",
			parsers.GameProfile, parsers.String.Behavior(parsers.GreedyPhrase),
		).Requires(3).Executes(banExecutor(s))
	})
}

func registerBanIP(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/ban-ip <target> [<reason>]",
			parsers.String, parsers.String.Behavior(parsers.GreedyPhrase),
		).Requires(3).Executes(banIPExecutor(s))
	})
}

func registerPardon(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/pardon <targets>", parsers.GameProfile).Requires(3).Executes(pardonExecutor(s))
	})
}

func registerPardonIP(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/pardon-ip <target>", parsers.String).Requires(3).Executes(pardonIPExecutor(s))
	})
}

func pardonExecutor(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetEntityTarget("targets")

		type unbanInfo struct {
			UUID string
			Name string
		}
		var unbans []unbanInfo

		switch target.Type {
		case mc.TargetTypeUUID:
			if banned, _ := s.PlayerRegistry.IsBanned(target.UUID); banned {
				s.PlayerRegistry.UnbanByUUID(target.UUID.String())
				unbans = append(unbans, unbanInfo{target.UUID.String(), target.UUID.String()})
			}
		case mc.TargetTypePlayerName:
			s.PlayerRegistry.Mu.RLock()
			var found bool
			for _, entry := range s.PlayerRegistry.BannedPlayers {
				if entry.Name == target.Name {
					found = true
					break
				}
			}
			s.PlayerRegistry.Mu.RUnlock()
			if found {
				s.PlayerRegistry.Unban(target.Name)
				unbans = append(unbans, unbanInfo{"", target.Name})
			}
		case mc.TargetTypeSelector:
			sourceUUID, sourcePos := commandSource(cc)
			resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
			for _, p := range resolved {
				if banned, _ := s.PlayerRegistry.IsBanned(uuid.UUID(p.UUID)); banned {
					s.PlayerRegistry.UnbanByUUID(uuid.UUID(p.UUID).String())
					unbans = append(unbans, unbanInfo{uuid.UUID(p.UUID).String(), p.Name})
				}
			}
		}

		if len(unbans) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsPardonFailed))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		for _, u := range unbans {
			name := u.Name
			if name == "" {
				name = u.UUID
			}
			cc.SendMessage(tc.Translatable(mcdata.CommandsPardonSuccess, tc.Text(name)))
		}

		return &CommandResult{Success: 1, Result: len(unbans)}, nil
	}
}

func pardonIPExecutor(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetString("target")

		if ip := net.ParseIP(target); ip == nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsPardonipInvalid))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		s.PlayerRegistry.Mu.RLock()
		var found bool
		for _, entry := range s.PlayerRegistry.BannedIPs {
			if entry.IP == target {
				found = true
				break
			}
		}
		s.PlayerRegistry.Mu.RUnlock()

		if !found {
			cc.SendMessage(tc.Translatable(mcdata.CommandsPardonipFailed))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		s.PlayerRegistry.UnbanIP(target)
		cc.SendMessage(tc.Translatable(mcdata.CommandsPardonipSuccess, tc.Text(target)))

		return &CommandResult{Success: 1, Result: 0}, nil
	}
}

func banExecutor(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetEntityTarget("targets")

		reason := "Banned by an operator."
		if cc.Args.Has("reason") {
			reason = cc.Args.GetString("reason")
		}

		targets := resolveProfileTargets(s, cc, target)
		if len(targets) == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		source := actorName(cc)
		affected := 0
		for _, t := range targets {
			if banned, _ := s.PlayerRegistry.IsBanned(t.UUID); banned {
				cc.SendMessage(tc.Translatable(mcdata.CommandsBanFailed))
				continue
			}
			s.PlayerRegistry.Ban(t.UUID, t.Name, source, reason, "forever")
			cc.SendMessage(tc.Translatable(mcdata.CommandsBanSuccess, tc.Text(t.Name), tc.Text(reason)))
			kickBannedPlayer(s, entity.NbtUUID(t.UUID), reason)
			affected++
		}

		if affected == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}
		return &CommandResult{Success: 1, Result: affected}, nil
	}
}

func banIPExecutor(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		target := cc.Args.GetString("target")

		reason := "Banned by an operator."
		if cc.Args.Has("reason") {
			reason = cc.Args.GetString("reason")
		}

		ip, ok := resolveIPTarget(s, target)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsBanipInvalid))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		s.PlayerRegistry.Mu.RLock()
		var alreadyBanned bool
		for _, entry := range s.PlayerRegistry.BannedIPs {
			if entry.IP == ip {
				alreadyBanned = true
				break
			}
		}
		s.PlayerRegistry.Mu.RUnlock()

		if alreadyBanned {
			cc.SendMessage(tc.Translatable(mcdata.CommandsBanipFailed))
			return &CommandResult{Success: 0, Result: 0}, nil
		}

		source := actorName(cc)
		s.PlayerRegistry.BanIP(ip, source, reason, "forever")
		cc.SendMessage(tc.Translatable(mcdata.CommandsBanipSuccess, tc.Text(ip), tc.Text(reason)))
		kickBannedIP(s, ip, reason)

		return &CommandResult{Success: 1, Result: 0}, nil
	}
}
