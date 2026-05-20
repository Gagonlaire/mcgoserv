package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/Tnze/go-mc/nbt"
)

func registerSummon(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/summon <entity> [<pos>] [<nbt>]",
			parsers.Resource(entity.Registry), parsers.Vec3, parsers.NbtCompoundTag,
		).Executes(func(cc *CommandContext) (*CommandResult, error) {
			sender := cc.Source.Entity.(*entity.Player)

			pos := sender.Position
			if cc.Args.Has("pos") {
				pos = cc.Args["pos"].(parsers.ParsedVec3).Resolve(sender.Position, sender.Rotation)
			}

			var compound nbt.StringifiedMessage
			if cc.Args.Has("nbt") {
				compound = cc.Args["nbt"].(nbt.StringifiedMessage)
			}

			return doSummon(s, cc, pos, compound)
		})
	})
}

func doSummon(s *server.Server, cc *CommandContext, pos [3]float64, compound nbt.StringifiedMessage) (*CommandResult, error) {
	entityID := GetArgument[entity.ID](cc.Args, "entity")

	sender := cc.Source.Entity.(*entity.Player)
	e := entity.NewFromType(entityID, sender.DimensionID, pos, sender.Rotation)
	if e == nil {
		cc.SendMessage(tc.Translatable(mcdata.CommandsSummonFailed).SetColor(tc.ColorRed))
		return &CommandResult{Success: 0}, nil
	}

	if compound != "" {
		merger, ok := e.(nbtpath.NbtTarget)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataEntityInvalid).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		v, err := nbtpath.SNBTToValue(compound)
		if err != nil {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		src, ok := v.(map[string]any)
		if !ok {
			cc.SendMessage(tc.Translatable(mcdata.CommandsDataMergeFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		if _, err := merger.NbtMerge(src); err != nil {
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
