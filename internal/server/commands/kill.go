package commands

import (
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
	"github.com/google/uuid"
)

func registerKill(s *server.Server) {
	s.Commander.Register(
		Literal("kill").Connect(
			Argument("target", parsers.Entity).
				Executes(func(cc *CommandContext) (*CommandResult, error) {
					sender, _ := cc.Source.Entity.(*entity.Player)
					target := cc.Args.GetEntityTarget("target")

					var senderUUID uuid.UUID
					var senderPos [3]float64
					if sender != nil {
						senderUUID = uuid.UUID(sender.UUID)
						senderPos = sender.Position
					}
					resolved := s.World.ResolveTarget(target, senderUUID, senderPos)

					if len(resolved) == 0 {
						cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
						return &CommandResult{Success: 0}, nil
					}

					var killer entity.Entity
					if sender != nil {
						killer = sender
					}
					for _, e := range resolved {
						s.Kill(e, killer, deathCause(e, sender))
					}

					count := len(resolved)
					if count == 1 {
						cc.SendMessage(tc.Translatable(mcdata.CommandsKillSuccessSingle, entityDisplayName(resolved[0])))
					} else {
						cc.SendMessage(tc.Translatable(mcdata.CommandsKillSuccessMultiple, tc.Text(strconv.Itoa(count))))
					}
					return &CommandResult{Success: 1, Result: count}, nil
				}),
		),
	)
}

func entityDisplayName(e entity.Entity) tc.Component {
	if player, ok := e.(*entity.Player); ok {
		return tc.PlayerName(player.Name)
	}
	return tc.Text(e.Base().ID.DisplayName())
}

func deathCause(target entity.Entity, killer *entity.Player) tc.Component {
	targetName := entityDisplayName(target)
	if killer != nil {
		return tc.Translatable(mcdata.DeathAttackGenericKillPlayer, targetName, tc.PlayerName(killer.Name))
	}
	return tc.Translatable(mcdata.DeathAttackGenericKill, targetName)
}
