package metadata

// https://minecraft.wiki/w/Java_Edition_protocol/Entity_metadata#Entity_Metadata_Format
const (
	TypeByte = iota
	TypeVarInt
	TypeVarLong
	TypeFloat
	TypeString
	TypeTextComponent
	TypeOptTextComponent
	TypeSlot
	TypeBoolean
	TypeRotations
	TypePosition
	TypeOptPosition
	TypeDirection
	TypeOptUUID
	TypeBlockState
	TypeOptBlockState
	TypeParticle
	TypeParticles
	TypeVillagerData
	TypeOptVarInt
	TypePose
	TypeCatVariant
	TypeCatSoundVariant
	TypeCowVariant
	TypeCowSoundVariant
	TypeWolfVariant
	TypeWolfSoundVariant
	TypeFrogVariant
	TypePigVariant
	TypePigSoundVariant
	TypeChickenVariant
	TypeChickenSoundVariant
	TypeZombieNautilusVariant
	TypeOptGlobalPosition
	TypePaintingVariant
	TypeSnifferState
	TypeArmadilloState
	TypeCopperGolemState
	TypeWeatheringCopperState
	TypeVector3
	TypeQuaternion
	TypeResolvableProfile
	TypeHumanoidArm
)

type Index = byte
