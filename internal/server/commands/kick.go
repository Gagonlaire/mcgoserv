package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func registerKick(s *server.Server) {
	s.Commander.Register(
		Literal("kick").Requires(0).Connect(
			Argument("targets", parsers.Entity.PlayersOnly(true)).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					player := cc.Source.Entity.(*entities.Player)
					targets := cc.Args.GetEntityTarget("targets")
					target := s.World.ResolveTarget(targets, player.UUID, player.Pos)
					targetConn, ok := s.ConnectionsByEID.Load(target[0].EntityID)

					if ok {
						targetConn.(*server.Connection).Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectKicked))
					}
					// TODO: replace with the correct message
					cc.SendMessage(tc.Translatable(
						mcdata.CommandsKickSuccess,
						tc.Text(targets.Name),
						tc.Translatable(mcdata.MultiplayerDisconnectKicked),
					))
					return &CommandResult{Success: 1, Result: 0}, nil
				}).
				Connect(
					Argument("reason", parsers.Message).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							_ = cc.Args.GetEntityTarget("targets")
							_ = cc.Args.GetString("reason")
							// TODO: resolve targets, disconnect matching players with reason
							return &CommandResult{Success: 1, Result: 0}, nil
						}),
				),
		),
	)
}
