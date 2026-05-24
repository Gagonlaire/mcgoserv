package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

func DecodeKeepAlive(pkt *packet.InboundPacket) (*proto.Long, error) {
	var id proto.Long
	if err := pkt.Decode(&id); err != nil {
		return nil, err
	}
	return &id, nil
}
