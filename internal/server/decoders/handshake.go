package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type Handshake struct {
	ServerAddress   proto.BoundedString
	ProtocolVersion proto.VarInt
	Intent          proto.VarInt
	ServerPort      proto.UnsignedShort
}

func DecodeHandshake(pkt *packet.InboundPacket) (*Handshake, error) {
	data := &Handshake{
		ServerAddress: proto.BoundedString{MaxLength: 255},
	}

	if err := pkt.Decode(&data.ProtocolVersion, &data.ServerAddress, &data.ServerPort, &data.Intent); err != nil {
		return nil, err
	}
	// todo: do something ?
	return data, nil
}
