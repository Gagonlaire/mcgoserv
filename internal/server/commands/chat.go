package commands

import (
	"context"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerMsg(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/msg <targets> <message ...>",
			parsers.Entity.PlayersOnly(true), parsers.Message,
		).Aliases("tell", "w").Executes(sendMsg(s))
	})
}

func registerTellRaw(s *server.Server) {
	s.Commander.Register(Literal("tellraw").Requires(2).Connect())
}

func registerSay(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/say <message ...>", parsers.Message).Requires(2).Executes(sayBroadcast(s))
	})
}

func registerTeamMsg(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/teammsg <message ...>", parsers.Message).Aliases("tm").
			Executes(func(cc *CommandContext) (*CommandResult, error) {
				panic(context.TODO())
			})
	})
}

func sendMsg(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		player := cc.Source.Entity.(*entity.Player)
		targets := cc.Args.GetEntityTarget("targets")
		message := cc.Args["message"].(*mc.ParsedMessage)
		text := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
		signature := cc.Signed.GetArgSignature("message")

		resolved := s.World.ResolvePlayers(targets, uuid.UUID(player.UUID), player.Position)
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

		if len(resolved) == 0 {
			return &CommandResult{Success: 0, Result: 0}, nil
		}
		return &CommandResult{Success: 1, Result: len(resolved)}, nil
	}
}

func sayBroadcast(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		player := cc.Source.Entity.(*entity.Player)
		message := cc.Args["message"].(*mc.ParsedMessage)
		text := s.World.ResolveMessage(message, uuid.UUID(player.UUID), player.Position)
		signature := cc.Signed.GetArgSignature("message")

		senderConn, ok := s.ConnectionsByEID.Load(player.EntityID)
		if !ok {
			return &CommandResult{Success: 0, Result: 0}, nil
		}
		sender := senderConn.(*server.Connection)

		recipients := 0
		for k := range s.Connections.Range {
			target := k.(*server.Connection)
			sender.SendSignedMessage(target, message.Raw, mc.Optional[tc.Component]{Present: true, Value: tc.Text(text)}, signature, cc.Signed, 5)
			recipients++
		}

		return &CommandResult{Success: 1, Result: recipients}, nil
	}
}
