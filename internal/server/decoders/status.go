package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func DecodePing(pkt *packet.Packet) (*mc.Long, error) {
	var timestamp mc.Long
	if err := pkt.Decode(&timestamp); err != nil {
		return nil, err
	}
	return &timestamp, nil
}
