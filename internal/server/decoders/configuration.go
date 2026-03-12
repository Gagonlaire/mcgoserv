package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

func DecodeServerboundKnownPacks(pkt *packet.InboundPacket) (*mc.PrefixedArray[mc.DataPackIdentifier, *mc.DataPackIdentifier], error) {
	var knownPacks mc.PrefixedArray[mc.DataPackIdentifier, *mc.DataPackIdentifier]
	if err := pkt.Decode(&knownPacks); err != nil {
		return nil, err
	}
	return &knownPacks, nil
}

func DecodeClientInformation(pkt *packet.InboundPacket) (*mc.ClientInformation, error) {
	var data mc.ClientInformation
	if err := pkt.Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}
