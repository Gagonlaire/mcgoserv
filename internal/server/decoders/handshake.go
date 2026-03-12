package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type Handshake struct {
	ProtocolVersion mc.VarInt
	ServerAddress   mc.String
	ServerPort      mc.UnsignedShort
	Intent          mc.VarInt
}

func DecodeHandshake(pkt *packet.Packet) (*Handshake, error) {
	data := &Handshake{}

	if err := pkt.Decode(&data.ProtocolVersion, &data.ServerAddress, &data.ServerPort, &data.Intent); err != nil {
		return nil, err
	}
	return data, nil
}
