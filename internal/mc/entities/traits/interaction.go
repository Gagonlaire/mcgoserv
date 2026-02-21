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

type Projectile interface {
	GetOwner() uuid.UUID
	SetOwner(uuid.UUID)
}
