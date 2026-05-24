package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

// todo: replace with a registry of all game events
const gameEventChangeGameMode = 3

var gameModeNames = [4]string{"survival", "creative", "adventure", "spectator"}

func registerGamemode(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/gamemode <mode> [<target>]",
			parsers.Gamemode, parsers.Entity.PlayersOnly(true),
		).Requires(2).Executes(setGamemode(s))
	})
}

func gameModeName(mode int32) string {
	if mode >= 0 && int(mode) < len(gameModeNames) {
		return gameModeNames[mode]
	}
	return "unknown"
}

func setGamemode(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		mode := GetArgument[int32](cc.Args, "mode")

		var targets []*entity.Player
		if cc.Args.Has("target") {
			sourceUUID, sourcePos := commandSource(cc)
			targets = s.World.ResolvePlayers(cc.Args.GetEntityTarget("target"), sourceUUID, sourcePos)
			if len(targets) == 0 {
				cc.SendMessage(tc.Translatable(mcdata.ArgumentEntityNotfoundPlayer).SetColor(tc.ColorRed))
				return &CommandResult{Success: 0}, nil
			}
		} else {
			self, ok := cc.Source.Entity.(*entity.Player)
			if !ok {
				return &CommandResult{Success: 0}, nil
			}
			targets = []*entity.Player{self}
		}

		modeComp := tc.Text(gameModeName(mode))
		self, _ := cc.Source.Entity.(*entity.Player)
		for _, p := range targets {
			applyGameMode(s, p, mode)
			if p == self {
				cc.SendMessage(tc.Translatable(mcdata.CommandsGamemodeSuccessSelf, modeComp))
			} else {
				cc.SendMessage(tc.Translatable(mcdata.CommandsGamemodeSuccessOther, tc.PlayerName(p.Name), modeComp))
			}
		}
		return &CommandResult{Success: 1, Result: len(targets)}, nil
	}
}

func applyGameMode(s *server.Server, p *entity.Player, mode int32) {
	p.PreviousGameMode = p.GameMode
	p.GameMode = mode
	if conn, ok := s.ConnectionsByEID.Load(p.EntityID); ok {
		c := conn.(*server.Connection)
		// todo: it must also update the player tab for everyone
		c.Send(c.NewPacket(packet.PlayClientboundGameEvent,
			proto.UnsignedByte(gameEventChangeGameMode), proto.Float(mode)))
	}
}
