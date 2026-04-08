package layers

//go:generate go run ../../../../cmd/gen-meta .

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
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

//meta:encode mode=layer
type AvatarData struct {
	BaseLayer
	MainHand  int32    `meta:"IndexMainHand,HumanoidArm"`
	SkinParts SkinPart `meta:"IndexSkinParts,Byte,flags"`
}
