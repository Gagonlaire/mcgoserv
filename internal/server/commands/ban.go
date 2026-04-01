package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func registerBan(s *server.Server) {
	s.Commander.Register(
		Literal("ban").Requires(3).Connect(
			Argument("targets", parsers.GameProfile).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					_ = cc.Args.GetEntityTarget("targets")
					// TODO: resolve targets, add to ban list, kick if online
					return &CommandResult{Success: 1, Result: 0}, nil
				}).
				Connect(
					Argument("reason", parsers.String.Behavior(parsers.GreedyPhrase)).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							_ = cc.Args.GetEntityTarget("targets")
							_ = cc.Args.GetString("reason")
							// TODO: resolve targets, add to ban list with reason, kick if online
							return &CommandResult{Success: 1, Result: 0}, nil
						}),
				),
		),

		Literal("ban-ip").Requires(3).Connect(
			Argument("target", parsers.String).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					_ = cc.Args.GetString("target")
					// TODO: validate target is a valid IP or online player name, add to IP ban list
					return &CommandResult{Success: 1, Result: 0}, nil
				}).
				Connect(
					Argument("reason", parsers.String.Behavior(parsers.GreedyPhrase)).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							_ = cc.Args.GetString("target")
							_ = cc.Args.GetString("reason")
							// TODO: validate target, add to IP ban list with reason, kick affected players
							return &CommandResult{Success: 1, Result: 0}, nil
						}),
				),
		),

		Literal("pardon").Requires(3).Connect(
			Argument("targets", parsers.String).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					_ = cc.Args.GetString("targets")
					// TODO: remove targets from ban list
					return &CommandResult{Success: 1, Result: 0}, nil
				}),
		),

		Literal("pardon-ip").Requires(3).Connect(
			Argument("target", parsers.String).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					_ = cc.Args.GetString("target")
					// TODO: remove target from IP ban list
					return &CommandResult{Success: 1, Result: 0}, nil
				}),
		),
	)
}
