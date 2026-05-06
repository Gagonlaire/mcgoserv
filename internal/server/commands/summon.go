package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/Tnze/go-mc/nbt"
)

type nbtMergeable interface {
	NbtMerge(compound nbt.StringifiedMessage) error
}

func registerSummon(s *server.Server) {
	s.Commander.Register(
		Literal("summon").Connect(
			Argument("entity", parsers.String).Executes(func(cc *CommandContext) (*CommandResult, error) {
				sender := cc.Source.Entity.(*entity.Player)
				return doSummon(s, cc, sender.Position, "")
			}).Connect(
				Argument("pos", parsers.Vec3).Executes(func(cc *CommandContext) (*CommandResult, error) {
					sender := cc.Source.Entity.(*entity.Player)
					pos := cc.Args["pos"].(parsers.ParsedVec3).Resolve(sender.Position, sender.Rotation)
					return doSummon(s, cc, pos, "")
				}).Connect(
					Argument("nbt", parsers.NbtCompoundTag).
						Executes(func(cc *CommandContext) (*CommandResult, error) {
							sender := cc.Source.Entity.(*entity.Player)
							pos := cc.Args["pos"].(parsers.ParsedVec3).Resolve(sender.Position, sender.Rotation)
							compound := cc.Args["nbt"].(nbt.StringifiedMessage)
							return doSummon(s, cc, pos, compound)
						}),
				),
			),
		),
	)
}

func doSummon(s *server.Server, cc *CommandContext, pos [3]float64, compound nbt.StringifiedMessage) (*CommandResult, error) {
	name := GetArgument[string](cc.Args, "entity")
	entityID, ok := entity.FromString(name)
	if !ok {
		cc.SendMessage(tc.Translatable(mcdata.CommandsSummonFailed).SetColor(tc.ColorRed))
		return &CommandResult{Success: 0}, nil
	}

	sender := cc.Source.Entity.(*entity.Player)
	e := entity.NewFromType(entityID, sender.DimensionID, pos, sender.Rotation)
	if e == nil {
		cc.SendMessage(tc.Translatable(mcdata.CommandsSummonFailed).SetColor(tc.ColorRed))
		return &CommandResult{Success: 0}, nil
	}

	if compound != "" {
		merger, ok := e.(nbtMergeable)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if err := merger.NbtMerge(compound); err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
	}

	if err := s.SpawnEntity(e); err != nil {
		cc.SendMessage(tc.Translatable(mcdata.CommandsSummonFailed).SetColor(tc.ColorRed))
		return &CommandResult{Success: 0}, nil
	}
	cc.SendMessage(tc.Translatable(mcdata.CommandsSummonSuccess, tc.Text(entityID.DisplayName())))
	return &CommandResult{Success: 1}, nil
}
