package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type LoginStart struct {
	Name       mc.String16
	PlayerUUID mc.UUID
}

type EncryptionResponse struct {
	EncryptedSecret      mc.PrefixedByteArray
	EncryptedVerifyToken mc.PrefixedByteArray
}

func DecodeLoginStart(pkt *packet.InboundPacket) (*LoginStart, error) {
	data := &LoginStart{}
	if err := pkt.Decode(&data.Name, &data.PlayerUUID); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeEncryptionResponse(pkt *packet.InboundPacket) (*EncryptionResponse, error) {
	data := &EncryptionResponse{
		EncryptedSecret:      mc.PrefixedByteArray{MaxLength: 128},
		EncryptedVerifyToken: mc.PrefixedByteArray{MaxLength: 128},
	}
	if err := pkt.Decode(&data.EncryptedSecret, &data.EncryptedVerifyToken); err != nil {
		return nil, err
	}
	return data, nil
}
