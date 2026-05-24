package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type LoginStart struct {
	Name       proto.String16
	PlayerUUID proto.UUID
}

type EncryptionResponse struct {
	EncryptedSecret      proto.PrefixedByteArray
	EncryptedVerifyToken proto.PrefixedByteArray
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
		EncryptedSecret:      proto.PrefixedByteArray{MaxLength: 128},
		EncryptedVerifyToken: proto.PrefixedByteArray{MaxLength: 128},
	}
	if err := pkt.Decode(&data.EncryptedSecret, &data.EncryptedVerifyToken); err != nil {
		return nil, err
	}
	return data, nil
}
