package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type SetPlayerPosition struct {
	X, Y, Z proto.Double
	Flags   proto.Byte
}
type SetPlayerRotation struct {
	Yaw, Pitch proto.Float
	Flags      proto.Byte
}

type SetPlayerPositionAndRotation struct {
	X, Y, Z    proto.Double
	Yaw, Pitch proto.Float
	Flags      proto.Byte
}

type CommandSuggestionsRequest struct {
	Text          proto.BoundedString
	TransactionID proto.VarInt
}

type ChatMessage struct {
	Signature    proto.PrefixedOptional[proto.ByteArray, *proto.ByteArray]
	Acknowledged proto.FixedBitSet
	Message      proto.String256
	Timestamp    proto.Long
	Salt         proto.Long
	MessageCount proto.VarInt
	Checksum     proto.Byte
}

type PlayerSession struct {
	PublicKey    proto.PrefixedByteArray
	KeySignature proto.PrefixedByteArray
	ExpiresAt    proto.Long
	SessionId    proto.UUID
}

type ArgumentSignature struct {
	ArgumentName proto.String16
	Signature    proto.ByteArray
}

type SignedChatCommand struct {
	Acknowledged       proto.FixedBitSet
	Command            proto.String
	ArgumentSignatures []ArgumentSignature
	Timestamp          proto.Long
	Salt               proto.Long
	MessageCount       proto.VarInt
	Checksum           proto.Byte
}

type PlayerCommand struct {
	EntityID, ActionID, JumpBoost proto.VarInt
}

type PlayerAction struct {
	Status   proto.VarInt
	Location proto.Position
	Face     proto.Byte
	Sequence proto.VarInt
}

type SetCreativeModeSlot struct {
	ClickedItem mc.Slot
	Slot        proto.Short
}

type UseItemOn struct {
	Hand                               proto.VarInt
	Location                           proto.Position
	Face                               proto.VarInt
	CursorPosX, CursorPosY, CursorPosZ proto.Float
	InsideBlock, WorldBorderHit        proto.Boolean
	Sequence                           proto.VarInt
}

func DecodeConfirmTeleportation(pkt *packet.InboundPacket) (*proto.VarInt, error) {
	var teleportId proto.VarInt
	if err := pkt.Decode(&teleportId); err != nil {
		return nil, err
	}
	return &teleportId, nil
}

func DecodeSetPlayerMovementFlags(pkt *packet.InboundPacket) (*proto.Byte, error) {
	var flags proto.Byte
	if err := pkt.Decode(&flags); err != nil {
		return nil, err
	}
	return &flags, nil
}

func DecodeSetPlayerPosition(pkt *packet.InboundPacket) (*SetPlayerPosition, error) {
	data := &SetPlayerPosition{}
	if err := pkt.Decode(&data.X, &data.Y, &data.Z, &data.Flags); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeSetPlayerRotation(pkt *packet.InboundPacket) (*SetPlayerRotation, error) {
	data := &SetPlayerRotation{}
	if err := pkt.Decode(&data.Yaw, &data.Pitch, &data.Flags); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeSetPlayerPositionAndRotation(pkt *packet.InboundPacket) (*SetPlayerPositionAndRotation, error) {
	data := &SetPlayerPositionAndRotation{}
	if err := pkt.Decode(
		&data.X, &data.Y, &data.Z,
		&data.Yaw, &data.Pitch,
		&data.Flags,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeCommandSuggestionsRequest(pkt *packet.InboundPacket) (*CommandSuggestionsRequest, error) {
	data := &CommandSuggestionsRequest{
		Text: proto.BoundedString{MaxLength: 32500},
	}
	if err := pkt.Decode(&data.TransactionID, &data.Text); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeChatMessage(pkt *packet.InboundPacket) (*ChatMessage, error) {
	data := &ChatMessage{
		Signature:    proto.NewPrefixedOptional[proto.ByteArray, *proto.ByteArray](proto.NewByteArray(256)),
		Acknowledged: proto.NewFixedBitSet(20),
	}
	if err := pkt.Decode(
		&data.Message,
		&data.Timestamp,
		&data.Salt,
		&data.Signature,
		&data.MessageCount,
		&data.Acknowledged,
		&data.Checksum,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodePlayerSession(pkt *packet.InboundPacket) (*PlayerSession, error) {
	data := &PlayerSession{
		PublicKey:    proto.PrefixedByteArray{MaxLength: 512},
		KeySignature: proto.PrefixedByteArray{MaxLength: 4096},
	}
	if err := pkt.Decode(
		&data.SessionId,
		&data.ExpiresAt,
		&data.PublicKey,
		&data.KeySignature,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeChatCommand(pkt *packet.InboundPacket) (*proto.String, error) {
	var command proto.String
	if err := pkt.Decode(&command); err != nil {
		return nil, err
	}
	return &command, nil
}

func DecodeSignedChatCommand(pkt *packet.InboundPacket) (*SignedChatCommand, error) {
	data := &SignedChatCommand{
		Acknowledged: proto.NewFixedBitSet(20),
	}
	var signaturesCount proto.VarInt

	_ = pkt.Decode(&data.Command, &data.Timestamp, &data.Salt, &signaturesCount)
	if err := pkt.Err(); err != nil {
		return nil, err
	}

	data.ArgumentSignatures = make([]ArgumentSignature, signaturesCount)
	for i := 0; i < int(signaturesCount); i++ {
		var argName proto.String16
		var signature = proto.NewByteArray(256)
		_ = pkt.Decode(&argName, &signature)
		data.ArgumentSignatures[i] = ArgumentSignature{
			ArgumentName: argName,
			Signature:    signature,
		}
	}
	_ = pkt.Decode(&data.MessageCount, &data.Acknowledged, &data.Checksum)

	if err := pkt.Err(); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodePlayerCommand(pkt *packet.InboundPacket) (*PlayerCommand, error) {
	data := &PlayerCommand{}
	if err := pkt.Decode(&data.EntityID, &data.ActionID, &data.JumpBoost); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodePlayerInput(pkt *packet.InboundPacket) (*proto.UnsignedByte, error) {
	var flags proto.UnsignedByte
	if err := pkt.Decode(&flags); err != nil {
		return nil, err
	}
	return &flags, nil
}

func DecodePlayerAction(pkt *packet.InboundPacket) (*PlayerAction, error) {
	data := &PlayerAction{}
	if err := pkt.Decode(
		&data.Status,
		&data.Location,
		&data.Face,
		&data.Sequence,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeSwingArm(pkt *packet.InboundPacket) (*proto.VarInt, error) {
	var hand proto.VarInt
	if err := pkt.Decode(&hand); err != nil {
		return nil, err
	}
	return &hand, nil
}

func DecodeSetHeldItem(pkt *packet.InboundPacket) (*proto.Short, error) {
	var slot proto.Short
	if err := pkt.Decode(&slot); err != nil {
		return nil, err
	}
	return &slot, nil
}

func DecodeSetCreativeModeSlot(pkt *packet.InboundPacket) (*SetCreativeModeSlot, error) {
	data := &SetCreativeModeSlot{}
	if err := pkt.Decode(&data.Slot, &data.ClickedItem); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeClientCommand(pkt *packet.InboundPacket) (*proto.VarInt, error) {
	var action proto.VarInt
	if err := pkt.Decode(&action); err != nil {
		return nil, err
	}
	return &action, nil
}

func DecodeUseItemOn(pkt *packet.InboundPacket) (*UseItemOn, error) {
	data := &UseItemOn{}
	if err := pkt.Decode(
		&data.Hand,
		&data.Location,
		&data.Face,
		&data.CursorPosX, &data.CursorPosY, &data.CursorPosZ,
		&data.InsideBlock,
		&data.WorldBorderHit,
		&data.Sequence,
	); err != nil {
		return nil, err
	}
	return data, nil
}
