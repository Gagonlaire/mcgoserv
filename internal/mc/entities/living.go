package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
)

const (
	IndexHandFlags      metadata.Index = 8
	IndexHealth         metadata.Index = 9
	IndexPotionColor    metadata.Index = 10
	IndexPotionAmbience metadata.Index = 11
	IndexArrowsInEntity metadata.Index = 12
	IndexBeeStingers    metadata.Index = 13
	IndexBedLocation    metadata.Index = 14
)

type HandState byte

const (
	HandStateIsHandActive   HandState = 0x01
	HandStateActiveHand     HandState = 0x02
	HandStateIsUsingRiptide HandState = 0x04
)

//meta:encode parents=BaseEntity
type LivingEntity struct {
	BaseEntity
	HandFlags           HandState                                      `meta:"IndexHandFlags,Byte,flags"`
	Health              float32                                        `meta:"IndexHealth,Float"`
	IsPotionAmbient     bool                                           `meta:"IndexPotionAmbience,Boolean"`
	ArrowsInEntity      int32                                          `meta:"IndexArrowsInEntity,VarInt"`
	BeeStingersInEntity int32                                          `meta:"IndexBeeStingers,VarInt"`
	BedLocation         mc.PrefixedOptional[mc.Position, *mc.Position] `meta:"IndexBedLocation,OptPosition"`
	// todo: Potion color Particles
}
