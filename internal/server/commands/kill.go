package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
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
					sender := cc.Source.Entity.(*entities.Player)
					target := cc.Args.GetEntityTarget("target")
					resolved := s.World.ResolveTarget(target, uuid.UUID(sender.UUID), sender.Position)

					if len(resolved) == 0 {
						cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundEntity).SetColor(tc.ColorRed))
						return &CommandResult{Success: 0}, nil
					}

					entity := resolved[0]
					displayName := entityDisplayName(entity)
					killEntity(s, entity)

					cc.SendMessage(tc.Translatable(mcdata.CommandsKillSuccessSingle, displayName))
					return &CommandResult{Success: 1, Result: 1}, nil
				}),
		),
	)
}

func killEntity(s *server.Server, entity entities.Entity) {
	if player, ok := entity.(*entities.Player); ok {
		if conn, loaded := s.ConnectionsByEID.Load(player.EntityID); loaded {
			conn.(*server.Connection).Disconnect(tc.Translatable(mcdata.CommandsKillSuccessSingle, tc.PlayerName(player.Name)))
			return
		}
		s.DespawnPlayer(player)
		return
	}
	s.DespawnEntity(entity)
}

func entityDisplayName(entity entities.Entity) tc.Component {
	if player, ok := entity.(*entities.Player); ok {
		return tc.PlayerName(player.Name)
	}
	if data := mcdata.GetEntity(int(entity.Base().TypeID)); data != nil {
		return tc.Text(data.DisplayName)
	}
	return tc.Text("entity")
}
