package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
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
				Argument("target", parsers.Entity.Single(true)).
					Executes(func(cc *CommandContext) (*CommandResult, error) {
						player := cc.Source.Entity.(*entities.Player)
						targets := cc.Args.GetEntityTarget("target")
						resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
						if len(resolved) == 0 {
							cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
							return &CommandResult{Success: 0}, nil
						}
						target := resolved[0]

						reader, ok := target.(nbtpath.NbtReader)
						if !ok {
							cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
							return &CommandResult{Success: 0}, nil
						}
						data, err := reader.NbtData()
						if err != nil {
							cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetUnknown, tc.Text(err.Error())).SetColor(tc.ColorRed))
							return &CommandResult{Success: 0}, nil
						}
						dataComp, err := nbtpath.SNBTToComponent(data)
						if err != nil {
							cc.SendMessage(tc.Translatable(mcdata.CommandsDataGetInvalid, tc.Text(err.Error())).SetColor(tc.ColorRed))
							return &CommandResult{Success: 0}, nil
						}

						cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityQuery, entityDisplayName(target), dataComp))
						return &CommandResult{Success: 1}, nil
					}),
			),

			Literal("merge").Connect(
				Argument("target", parsers.Entity.Single(true)).Connect(
					Argument("nbt", parsers.NbtCompoundTag).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							player := cc.Source.Entity.(*entities.Player)
							targets := cc.Args.GetEntityTarget("target")
							compound := GetArgument[nbt.StringifiedMessage](cc.Args, "nbt")
							resolved := s.World.ResolveTarget(targets, uuid.UUID(player.UUID), player.Position)
							if len(resolved) == 0 {
								cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
								return &CommandResult{Success: 0}, nil
							}
							target := resolved[0]

							merger, ok := target.(nbtMergeable)
							if !ok {
								cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
								return &CommandResult{Success: 0}, nil
							}
							if err := merger.NbtMerge(compound); err != nil {
								cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
								return &CommandResult{Success: 0}, nil
							}

							cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityModified, entityDisplayName(target)))
							return &CommandResult{Success: 1}, nil
						}),
				),
			),
		),
	)
}
