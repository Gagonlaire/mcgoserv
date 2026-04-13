package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerKick(s *server.Server) {
	s.Commander.Register(
		Literal("kick").Requires(3).Connect(
			Argument("targets", parsers.Entity.PlayersOnly(true)).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					player := cc.Source.Entity.(*entities.Player)
					targets := cc.Args.GetEntityTarget("targets")
					target := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
					targetConn, ok := s.ConnectionsByEID.Load(target[0].EntityID)

					if ok {
						cc.SendMessage(tc.Translatable(
							mcdata.CommandsKickSuccess,
							tc.Text(targets.Name),
							tc.Translatable(mcdata.MultiplayerDisconnectKicked),
						))
						targetConn.(*server.Connection).Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectKicked))
					}
					return &CommandResult{Success: 1, Result: 0}, nil
				}).
				Connect(
					Argument("reason", parsers.Message).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							player := cc.Source.Entity.(*entities.Player)
							targets := cc.Args.GetEntityTarget("targets")
							message := cc.Args["reason"].(*mc.ParsedMessage)
							kickMessage := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
							rTargets := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)

							cc.SendMessage(tc.Translatable(
								mcdata.CommandsKickSuccess,
								tc.Text(targets.Name),
								tc.Text(kickMessage),
							))
							for _, target := range rTargets {
								targetConn, ok := s.ConnectionsByEID.Load(target.EntityID)

								if ok {
									targetConn.(*server.Connection).Disconnect(tc.Text(kickMessage))
								}
							}
							return &CommandResult{Success: 1, Result: 0}, nil
						}),
				),
		),
	)
}
