package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/Tnze/go-mc/nbt"
	"github.com/google/uuid"
)

func registerData(s *server.Server) {
	// NOTE dummy impl for testing
	s.Commander.Register(
		Literal("data").Connect(
			Literal("get").Connect(
				Argument("target", parsers.Entity.PlayersOnly(true).Single(true)).
					Executes(func(cc *CommandContext) (*CommandResult, error) {
						player := cc.Source.Entity.(*entities.Player)
						targets := cc.Args.GetEntityTarget("target")
						resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
						if len(resolved) == 0 {
							cc.SendMessage(tc.Text("No entity found"))
							return &CommandResult{Success: 0}, nil
						}
						target := resolved[0]

						// todo: support direct entities
						data, err := target.Base().NbtData()
						if err != nil {
							cc.SendMessage(tc.Text("Error: " + err.Error()))
							return &CommandResult{Success: 0}, nil
						}

						cc.SendMessage(tc.Container(
							tc.Text(target.Name+" has the following entity data: ").SetColor(tc.ColorGreen),
							tc.Text(string(data)).SetColor(tc.ColorWhite),
						))
						return &CommandResult{Success: 1}, nil
					}),
			),

			Literal("merge").Connect(
				Argument("target", parsers.Entity.PlayersOnly(true).Single(true)).Connect(
					Argument("nbt", parsers.NbtCompoundTag).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							player := cc.Source.Entity.(*entities.Player)
							targets := cc.Args.GetEntityTarget("target")
							compound := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
							resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
							if len(resolved) == 0 {
								cc.SendMessage(tc.Text("No entity found"))
								return &CommandResult{Success: 0}, nil
							}
							target := resolved[0]

							// todo: support direct entities
							if err := target.Base().NbtMerge(compound); err != nil {
								cc.SendMessage(tc.Text("Error: " + err.Error()))
								return &CommandResult{Success: 0}, nil
							}

							cc.SendMessage(tc.Container(
								tc.Text("Merged NBT data into ").SetColor(tc.ColorGreen),
								tc.Text(target.Name).SetColor(tc.ColorWhite),
							))
							return &CommandResult{Success: 1}, nil
						}),
				),
			),
		),
	)
}
