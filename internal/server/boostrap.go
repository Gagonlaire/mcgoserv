package server

import (
	"strconv"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func (s *Server) registerTickerSteps() {
	s.Ticker.Register(func() { updateTime(s) })
	s.Ticker.Register(func() { processIncomingPackets(s) })
}

func (s *Server) registerCommands() {
	s.Commander.Register(
		Literal("stop").Executes(func(ctx *CommandContext) text_component.Component {
			server := ctx.Value("server").(*Server)
			server.Connections.Range(func(k, v interface{}) bool {
				conn := k.(*Connection)
				conn.Disconnect(text_component.Text("Server closed"))
				return true
			})
			// todo: server should actually stop :)
			server.Stop()
			return text_component.Text("Stopping the server")
		}),

		Literal("list").Executes(func(ctx *CommandContext) text_component.Component {
			server := ctx.Value("server").(*Server)
			players := make([]string, 0)
			playerList := text_component.Container()

			server.Connections.Range(func(k, v interface{}) bool {
				conn := k.(*Connection)

				if conn.Player != nil && conn.State == mc.StatePlay {
					// todo: check if listed
					players = append(players, string(conn.Player.Name))
					playerList.AddExtra(
						text_component.PresetPlayerName(string(conn.Player.Name)),
						text_component.Text(", "),
					)
				}
				return true
			})
			if len(players) > 0 {
				playerList.Extra = playerList.Extra[:len(playerList.Extra)-1]
			}

			return text_component.Translatable(
				mcdata.CommandsListPlayers,
				text_component.Text(strconv.Itoa(len(players))),
				text_component.Text(strconv.Itoa(server.Properties.MaxPlayers)),
				playerList,
			)
		}),
	)
}

func (s *Server) registerPacketHandlers() {
	s.Router.Register(mc.StateHandshake, packet.HandshakeServerboundIntention, (*Connection).HandleHandshakePacket)
	s.Router.Register(mc.StateStatus, packet.StatusServerboundStatusRequest, (*Connection).HandleStatusRequestPacket)
	s.Router.Register(mc.StateStatus, packet.StatusServerboundPingRequest, (*Connection).HandlePingPacket)
	s.Router.Register(mc.StateLogin, packet.LoginServerboundHello, (*Connection).HandleLoginStartPacket)
	s.Router.Register(mc.StateLogin, packet.LoginServerboundLoginAcknowledged, (*Connection).HandleLoginAckPacket)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundFinishConfiguration, (*Connection).HandleFinishConfigurationAckPacket)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundKeepAlive, (*Connection).HandleKeepAlivePacket)
	s.Router.Register(mc.StateConfiguration, packet.ConfigurationServerboundSelectKnownPacks, (*Connection).HandleClientKnownPacksPacket)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundAcceptTeleportation, (*Connection).HandleConfirmTeleportationPacket)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerPos, (*Connection).HandleMovePlayerPos)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerPosRot, (*Connection).HandleMovePlayerPosRot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerRot, (*Connection).HandleMovePlayerRot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundKeepAlive, (*Connection).HandleKeepAlivePacket)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundClientTickEnd, (*Connection).HandleClientTickEnd)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerLoaded, (*Connection).HandlePlayerLoaded)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundMovePlayerStatusOnly, (*Connection).HandleMovePlayerStatusOnly)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerCommand, (*Connection).HandlePlayerCommand)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerInput, (*Connection).HandlePlayerInput)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSwing, (*Connection).HandleSwingArm)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundPlayerAction, (*Connection).HandlePlayerAction)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChat, (*Connection).HandleChat)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundChatCommand, (*Connection).HandleChatCommand)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSetCarriedItem, (*Connection).HandleSetCarriedItem)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundSetCreativeModeSlot, (*Connection).HandleSetCreativeModeSlot)
	s.Router.Register(mc.StatePlay, packet.PlayServerboundUseItemOn, (*Connection).HandleUseItemOn)
}
