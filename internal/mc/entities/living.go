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
	SleepingPos []int `nbt:"sleeping_pos,omitempty"`
	HomePos     []int `nbt:"home_pos,omitempty"`
	BaseEntity
	BedLocation                mc.PrefixedOptional[mc.Position, *mc.Position] `meta:"IndexBedLocation,OptPosition" nbt:"-"`
	Health                     float32                                        `meta:"IndexHealth,Float,default=1.0"`
	HomeRadius                 float32                                        `nbt:"home_radius,omitempty"`
	TicksSinceLastHurtByMob    int32                                          `nbt:"ticks_since_last_hurt_by_mob,omitempty"`
	PotionEffectColor          int32                                          `meta:"IndexPotionColor,VarInt" nbt:"-"`
	LastHurtByPlayerMemoryTime int32                                          `nbt:"last_hurt_by_player_memory_time,omitempty"`
	ArrowsInEntity             int32                                          `meta:"IndexArrowsInEntity,VarInt" nbt:"-"`
	BeeStingersInEntity        int32                                          `meta:"IndexBeeStingers,VarInt" nbt:"-"`
	HurtByTimestamp            int32
	AbsorptionAmount           float32
	HurtTime                   int16
	DeathTime                  int16
	LastHurtByMob              NbtUUID `nbt:"last_hurt_by_mob,omitempty"`
	LastHurtByPlayer           NbtUUID `nbt:"last_hurt_by_player,omitempty"`
	LeftHanded                 bool
	NoAI                       bool `nbt:"omitempty"`
	PersistenceRequired        bool
	FallFlying                 bool
	HandFlags                  HandState `meta:"IndexHandFlags,Byte,flags" nbt:"-"`
	CanPickUpLoot              bool
	IsPotionAmbient            bool `meta:"IndexPotionAmbience,Boolean" nbt:"-"`
	// todo: implement active_effects, attributes, brain, drop_chances, equipment, leash, locator_bar_icon, team
	// todo: Tags common to all mobs with drops from loot tables
}
