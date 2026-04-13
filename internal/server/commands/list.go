package commands

import (
	"strconv"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/google/uuid"
)

func getPlayerList(srv *server.Server, withUUID bool) (tc.Component, int) {
	players := srv.World.Players()
	playerList := tc.Container()
	playerCount := 0

	for _, player := range players {
		if player.Information.AllowServerListings {
			var entry tc.Component

			if withUUID {
				entry = tc.Translatable(
					mcdata.CommandsListNameAndId,
					tc.Text(player.Name),
					tc.Text(uuid.UUID(player.UUID).String()),
				)
			} else {
				entry = tc.PlayerName(player.Name)
			}
			playerList.AddExtra(
				entry,
				tc.Text(", "),
			)
			playerCount++
		}
	}

	if len(playerList.Extra) > 0 {
		playerList.Extra = playerList.Extra[:len(playerList.Extra)-1]
	}

	return playerList, playerCount
}

func registerList(s *server.Server) {
	s.Commander.Register(
		Literal("list").Executes(func(cc *CommandContext) (*CommandResult, error) {
			playerList, playerCount := getPlayerList(s, false)

			cc.SendMessage(tc.Translatable(
				mcdata.CommandsListPlayers,
				tc.Text(strconv.Itoa(playerCount)),
				tc.Text(strconv.Itoa(s.Config.Server.MaxPlayers)),
				playerList,
			))

			return &CommandResult{Success: 1, Result: playerCount}, nil
		}).Connect(
			Literal("uuids").Executes(func(cc *CommandContext) (*CommandResult, error) {
				playerList, playerCount := getPlayerList(s, true)

				cc.SendMessage(tc.Translatable(
					mcdata.CommandsListPlayers,
					tc.Text(strconv.Itoa(playerCount)),
					tc.Text(strconv.Itoa(s.Config.Server.MaxPlayers)),
					playerList,
				))

				return &CommandResult{Success: 1, Result: playerCount}, nil
			}),
		),
	)
}
