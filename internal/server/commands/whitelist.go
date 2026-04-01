package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func registerWhitelist(s *server.Server) {
	s.Commander.Register(
		Literal("whitelist").Requires(3).Connect(
			Literal("add").Connect(
				Argument("targets", parsers.GameProfile).
					Executes(func(cc *CommandContext) (*CommandResult, error) {
						_ = cc.Args.GetEntityTarget("targets")
						// TODO: resolve targets, add to whitelist
						return &CommandResult{Success: 1, Result: 0}, nil
					}),
			),
			Literal("remove").Connect(
				Argument("targets", parsers.GameProfile).
					Executes(func(cc *CommandContext) (*CommandResult, error) {
						_ = cc.Args.GetEntityTarget("targets")
						// TODO: resolve targets, remove from whitelist
						return &CommandResult{Success: 1, Result: 0}, nil
					}),
			),
			Literal("list").Executes(func(cc *CommandContext) (*CommandResult, error) {
				// TODO: list whitelisted players
				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("on").Executes(func(cc *CommandContext) (*CommandResult, error) {
				// TODO: enable whitelist
				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("off").Executes(func(cc *CommandContext) (*CommandResult, error) {
				// TODO: disable whitelist
				return &CommandResult{Success: 1, Result: 0}, nil
			}),
			Literal("reload").Executes(func(cc *CommandContext) (*CommandResult, error) {
				// TODO: reload whitelist from disk
				return &CommandResult{Success: 1, Result: 0}, nil
			}),
		),
	)
}
