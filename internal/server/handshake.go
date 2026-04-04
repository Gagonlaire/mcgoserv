package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
)

func (c *Connection) HandleHandshake(data *decoders.Handshake) {
	if data.Intent == mc.VarInt(mc.StateStatus) || data.Intent == mc.VarInt(mc.StateLogin) {
		c.State = mc.State(data.Intent)
		if logger.IsDebug() {
			logger.Debug("Handshake from %s (protocol=%d, intent=%s)",
				logger.Network(c.Conn.RemoteAddr()),
				data.ProtocolVersion,
				mc.GetStateName(c.State),
			)
			if int(data.ProtocolVersion) != mcdata.ProtocolVersion {
				logger.Debug("Protocol mismatch: client=%d, server=%d",
					data.ProtocolVersion, mcdata.ProtocolVersion)
			}
		}
	}
}
