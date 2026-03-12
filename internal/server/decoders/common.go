package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func DecodeKeepAlive(pkt *packet.InboundPacket) (*mc.Long, error) {
	var id mc.Long
	if err := pkt.Decode(&id); err != nil {
		return nil, err
	}
	return &id, nil
}
