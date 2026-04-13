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
	HandStateNone           HandState = 0
	HandStateIsHandActive   HandState = 0x01
	HandStateActiveHand     HandState = 0x02
	HandStateIsUsingRiptide HandState = 0x04
)

//meta:encode mode=entity parents=BaseEntity
type LivingEntity struct {
	BaseEntity
	HandFlags                  HandState `meta:"IndexHandFlags,Byte,flags"`
	AbsorptionAmount           float32
	HurtByTimestamp            int32
	DeathTime                  int16
	HurtTime                   int16
	Health                     float32                                        `meta:"IndexHealth,Float,default=1.0"`
	PotionEffectColor          int32                                          `meta:"IndexPotionColor,VarInt"` // todo: this is supposed to be a Particles
	IsPotionAmbient            bool                                           `meta:"IndexPotionAmbience,Boolean"`
	ArrowsInEntity             int32                                          `meta:"IndexArrowsInEntity,VarInt"`
	BeeStingersInEntity        int32                                          `meta:"IndexBeeStingers,VarInt"`
	BedLocation                mc.PrefixedOptional[mc.Position, *mc.Position] `meta:"IndexBedLocation,OptPosition"`
	FallFlying                 bool
	LeftHanded                 bool
	NoAI                       bool `nbt:"omitempty"`
	PersistenceRequired        bool
	HomePos                    []int   `nbt:"home_pos,omitempty"` // todo: should maybe be optional, present for creakings or when a mob gets leashed
	HomeRadius                 float32 `nbt:"home_radius,omitempty"`
	SleepingPos                []int   `nbt:"sleeping_pos,omitempty"` // todo: should be optional
	CanPickUpLoot              bool
	LastHurtByMob              NbtUUID `nbt:"last_hurt_by_mob,omitempty"`
	LastHurtByPlayer           NbtUUID `nbt:"last_hurt_by_player,omitempty"`
	LastHurtByPlayerMemoryTime int32   `nbt:"last_hurt_by_player_memory_time,omitempty"` // exist when last_hurt_by_player exists and is valid
	TicksSinceLastHurtByMob    int32   `nbt:"ticks_since_last_hurt_by_mob,omitempty"`    // exist when last_hurt_by_mob exists and is valid
	// todo: implement active_effects, attributes, equipment, brain, drop_chances, leash, locator_bar_icon, Team
}
