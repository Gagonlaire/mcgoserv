package server

import (
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func (s *Server) registerTickerSteps() {
	s.Ticker.Register(func() { updateTime(s) })
	s.Ticker.Register(func() { processIncomingPackets(s) })
}

func (s *Server) registerCommands() {
	s.Commander.Register(
		Literal("stop").Executes(func(cc *CommandContext) (*CommandResult, error) {
			server := cc.Source.Server.(*Server)
			logger.Component(logger.INFO, tc.Text("Stopping the server"))
			server.Stop()

			return &CommandResult{Success: 1, Result: 0}, nil
		}),

		Literal("list").Executes(func(cc *CommandContext) (*CommandResult, error) {
			server := cc.Source.Server.(*Server)
			players := make([]string, 0)
			playerList := tc.Container()

			server.Connections.Range(func(k, v interface{}) bool {
				conn := k.(*Connection)

				if conn.Player != nil && conn.State == mc.StatePlay && conn.Player.Information.AllowServerListings {
					players = append(players, conn.Player.Name)
					playerList.AddExtra(
						tc.PlayerName(conn.Player.Name),
						tc.Text(", "),
					)
				}
				return true
			})
			if len(players) > 0 {
				playerList.Extra = playerList.Extra[:len(playerList.Extra)-1]
			}

			cc.SendMessage(tc.Translatable(
				mcdata.CommandsListPlayers,
				tc.Text(strconv.Itoa(len(players))),
				tc.Text(strconv.Itoa(server.Properties.MaxPlayers)),
				playerList,
			))

			return &CommandResult{Success: 1, Result: len(players)}, nil
		}),

		Literal("test").Connect(Argument("value", parsers.Message).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 0}, nil
		})),
	)
}

func (s *Server) registerPacketHandlers() {
	s.Router.Register(mc.StateHandshake, packet.HandshakeServerboundIntention, (*Connection).HandleHandshake)
	s.Router.Register(mc.StateStatus, packet.StatusServerboundStatusRequest, (*Connection).HandleStatusRequest)
	s.Router.Register(mc.StateStatus, packet.StatusServerboundPingRequest, (*Connection).HandlePing)
	s.Router.Register(mc.StateLogin, packet.LoginServerboundHello, (*Connection).HandleLoginStart)
	s.Router.Register(mc.StateLogin, packet.LoginServerboundLoginAcknowledged, (*Connection).HandleLoginAck)
	s.Router.Register(mc.StateLogin, packet.LoginServerboundKey, (*Connection).HandleLoginEncryptionResponse)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundFinishConfiguration, (*Connection).HandleFinishConfigurationAck)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundKeepAlive, (*Connection).HandleKeepAlive)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundSelectKnownPacks, (*Connection).HandleClientKnownPacks)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundClientInformation, (*Connection).HandleClientInformation)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundAcceptTeleportation, (*Connection).HandleConfirmTeleportation)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerPos, (*Connection).HandleMovePlayerPos)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerPosRot, (*Connection).HandleMovePlayerPosRot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerRot, (*Connection).HandleMovePlayerRot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundKeepAlive, (*Connection).HandleKeepAlive)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundClientTickEnd, (*Connection).HandleClientTickEnd)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerLoaded, (*Connection).HandlePlayerLoaded)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerStatusOnly, (*Connection).HandleMovePlayerStatusOnly)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerCommand, (*Connection).HandlePlayerCommand)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerInput, (*Connection).HandlePlayerInput)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSwing, (*Connection).HandleSwingArm)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerAction, (*Connection).HandlePlayerAction)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChat, (*Connection).HandleChat)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChatSessionUpdate, (*Connection).HandleChatSessionUpdate)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChatCommand, (*Connection).HandleChatCommand)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChatCommandSigned, (*Connection).HandleSignedChatCommand)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSetCarriedItem, (*Connection).HandleSetCarriedItem)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSetCreativeModeSlot, (*Connection).HandleSetCreativeModeSlot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundUseItemOn, (*Connection).HandleUseItemOn)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundClientInformation, (*Connection).HandleClientInformation)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundCommandSuggestion, (*Connection).HandleCommandSuggestion)
}
