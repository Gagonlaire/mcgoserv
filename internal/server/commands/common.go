package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/google/uuid"
)

// ProfileTarget is a flattened (UUID, Name) pair derived from an EntityTarget.
type ProfileTarget struct {
	UUID uuid.UUID
	Name string
}

// commandSource returns the canonical (UUID, position) of the command's executor.
// Non-player sources yield a zero UUID and zero position.
func commandSource(cc *CommandContext) (uuid.UUID, [3]float64) {
	if p, ok := cc.Source.Entity.(*entity.Player); ok {
		return uuid.UUID(p.UUID), p.Position
	}
	return uuid.UUID{}, [3]float64{}
}

func actorName(cc *CommandContext) string {
	if p, ok := cc.Source.Entity.(*entity.Player); ok {
		return p.Name
	}
	return "Server"
}

func entityDisplayName(e entity.Entity) tc.Component {
	if player, ok := e.(*entity.Player); ok {
		return tc.PlayerName(player.Name)
	}
	return tc.Text(e.Base().ID.DisplayName())
}

// resolveProfileTargets flattens a parsed GameProfile/EntitySelector target into a list of {UUID, Name} pairs.
func resolveProfileTargets(s *server.Server, cc *CommandContext, target *mc.EntityTarget) []ProfileTarget {
	switch target.Type {
	case mc.TargetTypeUUID:
		if s.Config.Security.OnlineMode {
			name, err := api.GetProfileNameByUUID(target.UUID)
			if err != nil {
				cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
				return nil
			}
			return []ProfileTarget{{target.UUID, name}}
		}
		return []ProfileTarget{{target.UUID, "Unknown"}}
	case mc.TargetTypePlayerName:
		if s.Config.Security.OnlineMode {
			u, realName, err := api.GetUserUUID(target.Name)
			if err != nil {
				cc.SendMessage(tc.Translatable(mcdata.ArgumentPlayerUnknown))
				return nil
			}
			return []ProfileTarget{{u, realName}}
		}
		return []ProfileTarget{{api.OfflineUUID(target.Name), target.Name}}
	case mc.TargetTypeSelector:
		sourceUUID, sourcePos := commandSource(cc)
		resolved := s.World.ResolvePlayers(target, sourceUUID, sourcePos)
		out := make([]ProfileTarget, 0, len(resolved))
		for _, p := range resolved {
			out = append(out, ProfileTarget{uuid.UUID(p.UUID), p.Name})
		}
		return out
	}
	return nil
}
