package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

func DecodePing(pkt *packet.InboundPacket) (*proto.Long, error) {
	var timestamp proto.Long
	if err := pkt.Decode(&timestamp); err != nil {
		return nil, err
	}
	return &timestamp, nil
}
