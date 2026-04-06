package traits

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/google/uuid"
)

type Damageable interface {
	Kill()
}

type Bucketable interface {
	FromBucket() bool
	SetFromBucket(bool)
	GetBucketItem() mcdata.Item
}

type Saddleable interface {
	IsSaddled() bool
	Saddle(sound bool)
}

type HasProjectileOwner interface {
	GetOwner() uuid.UUID
	SetOwner(uuid.UUID)
}

type Tameable interface {
	IsTamed() bool
	GetOwnerUUID() uuid.UUID
}

type Ageable interface {
	GetAge() int32
	SetAge(int32)
	IsBaby() bool
}

type Breedable interface {
	Ageable
	IsInLove() bool
	SetInLove(ticks int32)
}
