package layers

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

const (
	IndexMainHand  metadata.Index = 15
	IndexSkinParts metadata.Index = 16
)

type SkinPart byte

const (
	SkinPartCape        SkinPart = 0x01
	SkinPartJacket      SkinPart = 0x02
	SkinPartLeftSleeve  SkinPart = 0x04
	SkinPartRightSleeve SkinPart = 0x08
	SkinPartLeftPants   SkinPart = 0x10
	SkinPartRightPants  SkinPart = 0x20
	SkinPartHat         SkinPart = 0x40
)

type AvatarData struct {
	BaseLayer
	MainHand  int32
	SkinParts SkinPart
}

func (a *AvatarData) SetMainHand(hand int32) {
	if a.MainHand != hand {
		a.MainHand = hand
		a.markDirty(IndexMainHand)
	}
}

func (a *AvatarData) SetSkinParts(parts SkinPart) {
	if a.SkinParts != parts {
		a.SkinParts = parts
		a.markDirty(IndexSkinParts)
	}
}

func (a *AvatarData) SetSkinPart(part SkinPart, on bool) {
	var newParts SkinPart
	if on {
		newParts = a.SkinParts | part
	} else {
		newParts = a.SkinParts &^ part
	}
	a.SetSkinParts(newParts)
}

func (a *AvatarData) EncodeMetadata(pkt *packet.OutboundPacket) {
	if a.isDirty(IndexMainHand) {
		_ = pkt.Encode(mc.UnsignedByte(IndexMainHand), mc.VarInt(metadata.TypeHumanoidArm), mc.Byte(a.MainHand))
	}
	if a.isDirty(IndexSkinParts) {
		_ = pkt.Encode(mc.UnsignedByte(IndexSkinParts), mc.VarInt(metadata.TypeByte), mc.Byte(a.SkinParts))
	}
}
