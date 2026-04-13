package commands

import (
	"context"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerMsg(s *server.Server) {
	s.Commander.Register(Literal("msg").Connect(
		Argument("targets", parsers.Entity.PlayersOnly(true)).Connect(
			Argument("message", parsers.Message).Executes(func(cc *CommandContext) (*CommandResult, error) {
				// todo: it break at the third message
				player := cc.Source.Entity.(*entities.Player)
				targets := cc.Args.GetEntityTarget("targets")
				message := cc.Args["message"].(*mc.ParsedMessage)
				text := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
				signature := cc.Signed.GetArgSignature("message")

				resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
				senderConn, ok := s.ConnectionsByEID.Load(player.EntityID)
				if !ok {
					return &CommandResult{Success: 0, Result: 0}, nil
				}
				sender := senderConn.(*server.Connection)
				sender.SendSignedMessage(sender, message.Raw, mc.Optional[tc.Component]{Present: true, Value: tc.Text(text)}, signature, cc.Signed, 4)

				for _, target := range resolved {
					targetConn, ok := s.ConnectionsByEID.Load(target.EntityID)
					if !ok {
						continue
					}
					receiver := targetConn.(*server.Connection)
					sender.SendSignedMessage(receiver, message.Raw, mc.Optional[tc.Component]{Present: true, Value: tc.Text(text)}, signature, cc.Signed, 3)
				}

				return &CommandResult{Success: len(resolved), Result: 0}, nil
			}),
		),
	))

	msg := s.Commander.Resolve("msg")
	s.Commander.Register(Literal("tell").RedirectTo(msg))
	s.Commander.Register(Literal("w").RedirectTo(msg))
}

func registerTellRaw(s *server.Server) {
	s.Commander.Register(Literal("tellraw").Connect())
}

func registerSay(s *server.Server) {
	s.Commander.Register(Literal("say").Connect(
		Argument("message", parsers.Message).Executes(func(cc *CommandContext) (*CommandResult, error) {
			player := cc.Source.Entity.(*entities.Player)
			message := cc.Args["message"].(*mc.ParsedMessage)
			text := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
			signature := cc.Signed.GetArgSignature("message")

			senderConn, ok := s.ConnectionsByEID.Load(player.EntityID)
			if !ok {
				return &CommandResult{Success: 0, Result: 0}, nil
			}
			sender := senderConn.(*server.Connection)

			for k := range s.Connections.Range {
				target := k.(*server.Connection)
				sender.SendSignedMessage(target, message.Raw, mc.Optional[tc.Component]{Present: true, Value: tc.Text(text)}, signature, cc.Signed, 5)
			}

			return &CommandResult{Success: 0, Result: 0}, nil
		}),
	))
}

func registerTeamMsg(s *server.Server) {
	s.Commander.Register(Literal("teammsg").Connect(
		Argument("message", parsers.Message).Executes(func(cc *CommandContext) (*CommandResult, error) {
			panic(context.TODO())
		}),
	))

	teamMsg := s.Commander.Resolve("teammsg")
	s.Commander.Register(Literal("tm").RedirectTo(teamMsg))
}
