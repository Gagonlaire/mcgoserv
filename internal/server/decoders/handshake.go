package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Handshake struct {
	ServerAddress   mc.BoundedString
	ProtocolVersion mc.VarInt
	Intent          mc.VarInt
	ServerPort      mc.UnsignedShort
}

func DecodeHandshake(pkt *packet.InboundPacket) (*Handshake, error) {
	data := &Handshake{
		ServerAddress: mc.BoundedString{MaxLength: 255},
	}

	if err := pkt.Decode(&data.ProtocolVersion, &data.ServerAddress, &data.ServerPort, &data.Intent); err != nil {
		return nil, err
	}
	// todo: do something ?
	return data, nil
}
