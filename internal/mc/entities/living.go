package entities

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities/metadata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
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

type LivingEntity struct {
	BaseEntity
	HandFlags           HandState
	Health              float32
	IsPotionAmbient     bool
	ArrowsInEntity      int32
	BeeStingersInEntity int32
	BedLocation         mc.PrefixedOptional[mc.Position, *mc.Position]
	// todo: Potion color Particles
}

func (l *LivingEntity) SetHandFlags(flags HandState) {
	if l.HandFlags != flags {
		l.HandFlags = flags
		l.MarkDirty(IndexHandFlags)
	}
}

func (l *LivingEntity) SetHandFlag(flag HandState, on bool) {
	var newFlags HandState
	if on {
		newFlags = l.HandFlags | flag
	} else {
		newFlags = l.HandFlags &^ flag
	}
	l.SetHandFlags(newFlags)
}

func (l *LivingEntity) SetHealth(health float32) {
	if l.Health != health {
		l.Health = health
		l.MarkDirty(IndexHealth)
	}
}

func (l *LivingEntity) SetPotionAmbient(ambient bool) {
	if l.IsPotionAmbient != ambient {
		l.IsPotionAmbient = ambient
		l.MarkDirty(IndexPotionAmbience)
	}
}

func (l *LivingEntity) SetArrowsInEntity(count int32) {
	if l.ArrowsInEntity != count {
		l.ArrowsInEntity = count
		l.MarkDirty(IndexArrowsInEntity)
	}
}

func (l *LivingEntity) SetBeeStingersInEntity(count int32) {
	if l.BeeStingersInEntity != count {
		l.BeeStingersInEntity = count
		l.MarkDirty(IndexBeeStingers)
	}
}

func (l *LivingEntity) SetBedLocation(m mc.PrefixedOptional[mc.Position, *mc.Position]) {
	if l.BedLocation != m {
		l.BedLocation = m
		l.MarkDirty(IndexBedLocation)
	}
}

func (l *LivingEntity) EncodeMetadata(pkt *packet.OutboundPacket) {
	l.BaseEntity.EncodeMetadata(pkt)

	if l.DirtyTracker.IsDirty(IndexHandFlags) {
		_ = pkt.Encode(mc.UnsignedByte(IndexHandFlags), mc.VarInt(metadata.TypeByte), mc.Byte(l.HandFlags))
	}
	if l.DirtyTracker.IsDirty(IndexHealth) {
		_ = pkt.Encode(mc.UnsignedByte(IndexHealth), mc.VarInt(metadata.TypeFloat), mc.Float(l.Health))
	}
	if l.DirtyTracker.IsDirty(IndexPotionAmbience) {
		_ = pkt.Encode(mc.UnsignedByte(IndexPotionAmbience), mc.VarInt(metadata.TypeBoolean), mc.Boolean(l.IsPotionAmbient))
	}
	if l.DirtyTracker.IsDirty(IndexArrowsInEntity) {
		_ = pkt.Encode(mc.UnsignedByte(IndexArrowsInEntity), mc.VarInt(metadata.TypeVarInt), mc.VarInt(l.ArrowsInEntity))
	}
	if l.DirtyTracker.IsDirty(IndexBeeStingers) {
		_ = pkt.Encode(mc.UnsignedByte(IndexBeeStingers), mc.VarInt(metadata.TypeVarInt), mc.VarInt(l.BeeStingersInEntity))
	}
	if l.DirtyTracker.IsDirty(IndexBedLocation) {
		_ = pkt.Encode(mc.UnsignedByte(IndexBedLocation), mc.VarInt(metadata.TypeOptPosition), l.BedLocation)
	}
}
