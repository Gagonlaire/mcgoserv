package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerKick(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/kick <targets> [<reason>]",
			parsers.Entity.PlayersOnly(true), parsers.Message,
		).Requires(3).Executes(func(cc *CommandContext) (*CommandResult, error) {
			player := cc.Source.Entity.(*entity.Player)
			targets := cc.Args.GetEntityTarget("targets")

			var reason, disconnect tc.Component = tc.Translatable(mcdata.MultiplayerDisconnectKicked), tc.Translatable(mcdata.MultiplayerDisconnectKicked)
			// todo: factor out the disconnect pattern
			if cc.Args.Has("reason") {
				message := cc.Args["reason"].(*mc.ParsedMessage)
				resolved := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
				reason = tc.Text(resolved)
				disconnect = tc.Text(resolved)
			}

			rTargets := s.World.ResolvePlayers(targets, uuid.UUID(player.UUID), player.Position)
			cc.SendMessage(tc.Translatable(mcdata.CommandsKickSuccess, tc.Text(targets.Name), reason))
			for _, target := range rTargets {
				if targetConn, ok := s.ConnectionsByEID.Load(target.EntityID); ok {
					targetConn.(*server.Connection).Disconnect(disconnect)
				}
			}
			return &CommandResult{Success: 1, Result: 0}, nil
		})
	})
}
