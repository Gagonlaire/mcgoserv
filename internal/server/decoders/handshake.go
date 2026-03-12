package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Handshake struct {
	ServerAddress   mc.String
	ProtocolVersion mc.VarInt
	Intent          mc.VarInt
	ServerPort      mc.UnsignedShort
}

func DecodeHandshake(pkt *packet.InboundPacket) (*Handshake, error) {
	data := &Handshake{}

	if err := pkt.Decode(&data.ProtocolVersion, &data.ServerAddress, &data.ServerPort, &data.Intent); err != nil {
		return nil, err
	}
	return data, nil
}
