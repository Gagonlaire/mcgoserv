package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
)

func (s *Server) registerTickerSteps() {
	s.Ticker.Register(func() { updateTime(s) })
	s.Ticker.Register(func() { processIncomingPackets(s) })
}

func (s *Server) registerPacketHandlers() {
	registerHandshakeHandlers(s)
	registerStatusHandlers(s)
	registerLoginHandlers(s)
	registerConfigurationHandlers(s)
	registerPlayHandlers(s)
}

func registerHandshakeHandlers(s *Server) {
	RegisterTyped(
		s.Router,
		mc.StateHandshake, packet.HandshakeServerboundIntention,
		false,
		decoders.DecodeHandshake, (*Connection).HandleHandshake,
	)
}

func registerStatusHandlers(s *Server) {
	RegisterRaw(
		s.Router,
		mc.StateStatus, packet.StatusServerboundStatusRequest,
		false,
		(*Connection).HandleStatusRequest,
	)
	RegisterTyped(
		s.Router,
		mc.StateStatus, packet.StatusServerboundPingRequest,
		false,
		decoders.DecodePing, (*Connection).HandlePing,
	)
}

func registerLoginHandlers(s *Server) {
	RegisterTyped(
		s.Router,
		mc.StateLogin, packet.LoginServerboundHello,
		false,
		decoders.DecodeLoginStart, (*Connection).HandleLoginStart,
	)
	RegisterTyped(
		s.Router,
		mc.StateLogin, packet.LoginServerboundKey,
		false,
		decoders.DecodeEncryptionResponse, (*Connection).HandleEncryptionResponse,
	)
	RegisterRaw(
		s.Router,
		mc.StateLogin, packet.LoginServerboundLoginAcknowledged,
		false,
		(*Connection).HandleLoginAcknowledged,
	)
}

func registerConfigurationHandlers(s *Server) {
	RegisterTyped(
		s.Router,
		mc.StateConfiguration, packet.ConfigurationServerboundClientInformation,
		false,
		decoders.DecodeClientInformation, (*Connection).HandleClientInformation,
	)
	RegisterRaw(
		s.Router,
		mc.StateConfiguration, packet.ConfigurationServerboundFinishConfiguration,
		false,
		(*Connection).HandleAcknowledgeFinishConfiguration,
	)
	RegisterTyped(
		s.Router,
		mc.StateConfiguration, packet.ConfigurationServerboundKeepAlive,
		false,
		decoders.DecodeKeepAlive, (*Connection).HandleKeepAlive,
	)
	RegisterTyped(
		s.Router,
		mc.StateConfiguration, packet.ConfigurationServerboundSelectKnownPacks,
		false,
		decoders.DecodeServerboundKnownPacks, (*Connection).HandleServerboundKnownPacks,
	)
}

func registerPlayHandlers(s *Server) {
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundAcceptTeleportation,
		true,
		decoders.DecodeConfirmTeleportation, (*Connection).HandleConfirmTeleportation,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundChatCommand,
		true,
		decoders.DecodeChatCommand, (*Connection).HandleChatCommand,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundChatCommandSigned,
		true,
		decoders.DecodeSignedChatCommand, (*Connection).HandleSignedChatCommand,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundChat,
		true,
		decoders.DecodeChatMessage, (*Connection).HandleChatMessage,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundChatSessionUpdate,
		true,
		decoders.DecodePlayerSession, (*Connection).HandlePlayerSession,
	)
	RegisterIgnored(
		s.Router,
		mc.StatePlay, packet.PlayServerboundClientTickEnd,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundClientInformation,
		true,
		decoders.DecodeClientInformation, (*Connection).HandleClientInformation,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundCommandSuggestion,
		true,
		decoders.DecodeCommandSuggestionsRequest, (*Connection).HandleCommandSuggestion,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundKeepAlive,
		false,
		decoders.DecodeKeepAlive, (*Connection).HandleKeepAlive,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundMovePlayerPos,
		true,
		decoders.DecodeSetPlayerPosition, (*Connection).HandleSetPlayerPosition,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundMovePlayerPosRot,
		true,
		decoders.DecodeSetPlayerPositionAndRotation, (*Connection).HandleSetPlayerPositionAndRotation,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundMovePlayerRot,
		true,
		decoders.DecodeSetPlayerRotation, (*Connection).HandleSetPlayerRotation,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundMovePlayerStatusOnly,
		true,
		decoders.DecodeSetPlayerMovementFlags, (*Connection).HandleSetPlayerMovementFlags,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundPlayerAction,
		true,
		decoders.DecodePlayerAction, (*Connection).HandlePlayerAction,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundPlayerCommand,
		true,
		decoders.DecodePlayerCommand, (*Connection).HandlePlayerCommand,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay,
		packet.PlayServerboundPlayerInput,
		true,
		decoders.DecodePlayerInput, (*Connection).HandlePlayerInput,
	)
	RegisterRaw(
		s.Router,
		mc.StatePlay, packet.PlayServerboundPlayerLoaded,
		true,
		(*Connection).HandlePlayerLoaded,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundSetCarriedItem,
		true,
		decoders.DecodeSetHeldItem, (*Connection).HandleSetHeldItem,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundSetCreativeModeSlot,
		true,
		decoders.DecodeSetCreativeModeSlot, (*Connection).HandleSetCreativeModeSlot,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundSwing,
		true,
		decoders.DecodeSwingArm, (*Connection).HandleSwingArm,
	)
	RegisterTyped(
		s.Router,
		mc.StatePlay, packet.PlayServerboundUseItemOn,
		true,
		decoders.DecodeUseItemOn, (*Connection).HandleUseItemOn,
	)
}
