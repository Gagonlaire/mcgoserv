package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type LoginStart struct {
	Name       mc.String
	PlayerUUID mc.UUID
}

type EncryptionResponse struct {
	EncryptedSecret      mc.PrefixedArray[mc.Byte, *mc.Byte]
	EncryptedVerifyToken mc.PrefixedArray[mc.Byte, *mc.Byte]
}

func DecodeLoginStart(pkt *packet.InboundPacket) (*LoginStart, error) {
	data := &LoginStart{}
	if err := pkt.Decode(&data.Name, &data.PlayerUUID); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeEncryptionResponse(pkt *packet.InboundPacket) (*EncryptionResponse, error) {
	data := &EncryptionResponse{}
	if err := pkt.Decode(&data.EncryptedSecret, &data.EncryptedVerifyToken); err != nil {
		return nil, err
	}
	return data, nil
}
