package decoders

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type SetPlayerPosition struct {
	X, Y, Z mc.Double
	Flags   mc.Byte
}
type SetPlayerRotation struct {
	Yaw, Pitch mc.Float
	Flags      mc.Byte
}

type SetPlayerPositionAndRotation struct {
	X, Y, Z    mc.Double
	Yaw, Pitch mc.Float
	Flags      mc.Byte
}

type CommandSuggestionsRequest struct {
	TransactionID mc.VarInt
	Text          mc.String
}

type ChatMessage struct {
	Message         mc.String
	Timestamp, Salt mc.Long
	Signature       mc.PrefixedOptional[mc.Array[mc.Byte, *mc.Byte], *mc.Array[mc.Byte, *mc.Byte]]
	MessageCount    mc.VarInt
	Acknowledged    *mc.FixedBitSet
	Checksum        mc.Byte
}

type PlayerSession struct {
	SessionId               mc.UUID
	ExpiresAt               mc.Long
	PublicKey, KeySignature mc.PrefixedArray[mc.Byte, *mc.Byte]
}

type ArgumentSignature struct {
	ArgumentName mc.String
	Signature    mc.Array[mc.Byte, *mc.Byte]
}

type SignedChatCommand struct {
	Command            mc.String
	Timestamp, Salt    mc.Long
	ArgumentSignatures []ArgumentSignature
	MessageCount       mc.VarInt
	Acknowledged       *mc.FixedBitSet
	Checksum           mc.Byte
}

type PlayerCommand struct {
	EntityID, ActionID, JumpBoost mc.VarInt
}

type PlayerAction struct {
	Status   mc.VarInt
	Location mc.Position
	Face     mc.Byte
	Sequence mc.VarInt
}

type SetCreativeModeSlot struct {
	Slot        mc.Short
	ClickedItem mc.Slot
}

type UseItemOn struct {
	Hand                               mc.VarInt
	Location                           mc.Position
	Face                               mc.VarInt
	CursorPosX, CursorPosY, CursorPosZ mc.Float
	InsideBlock, WorldBorderHit        mc.Boolean
	Sequence                           mc.VarInt
}

func DecodeConfirmTeleportation(pkt *packet.InboundPacket) (*mc.VarInt, error) {
	var teleportId mc.VarInt
	if err := pkt.Decode(&teleportId); err != nil {
		return nil, err
	}
	return &teleportId, nil
}

func DecodeSetPlayerMovementFlags(pkt *packet.InboundPacket) (*mc.Byte, error) {
	var flags mc.Byte
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
	data := &CommandSuggestionsRequest{}
	if err := pkt.Decode(&data.TransactionID, &data.Text); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodeChatMessage(pkt *packet.InboundPacket) (*ChatMessage, error) {
	arr := mc.NewArray[mc.Byte, *mc.Byte](256)
	data := &ChatMessage{
		Signature:    mc.NewPrefixedOptional(&arr),
		Acknowledged: mc.NewFixedBitSet(20),
	}
	if err := pkt.Decode(
		&data.Message,
		&data.Timestamp,
		&data.Salt,
		&data.Signature,
		&data.MessageCount,
		data.Acknowledged,
		&data.Checksum,
	); err != nil {
		return nil, err
	}
	return data, nil
}

func DecodePlayerSession(pkt *packet.InboundPacket) (*PlayerSession, error) {
	data := &PlayerSession{}
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

func DecodeChatCommand(pkt *packet.InboundPacket) (*mc.String, error) {
	var command mc.String
	if err := pkt.Decode(&command); err != nil {
		return nil, err
	}
	return &command, nil
}

func DecodeSignedChatCommand(pkt *packet.InboundPacket) (*SignedChatCommand, error) {
	data := &SignedChatCommand{
		Acknowledged: mc.NewFixedBitSet(20),
	}
	var signaturesCount mc.VarInt

	if err := pkt.Decode(&data.Command, &data.Timestamp, &data.Salt, &signaturesCount); err != nil {
		return nil, err
	}
	// todo: implement ReaderFrom for ArgumentSignature: use a prefixed array instead
	data.ArgumentSignatures = make([]ArgumentSignature, signaturesCount)
	for i := 0; i < int(signaturesCount); i++ {
		var argName mc.String
		var signature = mc.NewArray[mc.Byte, *mc.Byte](256)
		if err := pkt.Decode(&argName, &signature); err != nil {
			return nil, err
		}
		data.ArgumentSignatures[i] = ArgumentSignature{
			ArgumentName: argName,
			Signature:    signature,
		}
	}
	if err := pkt.Decode(&data.MessageCount, data.Acknowledged, &data.Checksum); err != nil {
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

func DecodePlayerInput(pkt *packet.InboundPacket) (*mc.UnsignedByte, error) {
	var flags mc.UnsignedByte
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

func DecodeSwingArm(pkt *packet.InboundPacket) (*mc.VarInt, error) {
	var hand mc.VarInt
	if err := pkt.Decode(&hand); err != nil {
		return nil, err
	}
	return &hand, nil
}

func DecodeSetHeldItem(pkt *packet.InboundPacket) (*mc.Short, error) {
	var slot mc.Short
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
