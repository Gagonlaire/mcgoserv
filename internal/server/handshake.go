package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
)

func (c *Connection) HandleHandshake(data *decoders.Handshake) {
	if data.Intent == mc.VarInt(mc.StateStatus) || data.Intent == mc.VarInt(mc.StateLogin) {
		c.State = mc.State(data.Intent)
	}
}
