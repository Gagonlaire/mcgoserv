package encoders

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type AddEntity struct {
	EntityID proto.VarInt
	UUID     proto.UUID
	TypeID   proto.VarInt
	Pos      proto.Coordinate
	Motion   proto.LpVec3
	Pitch    proto.Angle
	Yaw      proto.Angle
	HeadYaw  proto.Angle
	Data     proto.VarInt
}

func (a *AddEntity) WriteTo(w io.Writer) (n int64, err error) {
	fields := [9]io.WriterTo{
		a.EntityID, a.UUID, a.TypeID,
		a.Pos, &a.Motion,
		a.Pitch, a.Yaw, a.HeadYaw,
		a.Data,
	}
	for _, f := range fields {
		nn, err := f.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func NewAddEntity(entity entity.Entity) *AddEntity {
	rot := entity.GetRot()
	motion := entity.GetMotion()
	yaw := proto.DegreesToAngle(rot[0])
	return &AddEntity{
		EntityID: proto.VarInt(entity.GetID()),
		UUID:     proto.UUID(entity.GetUUID()),
		TypeID:   proto.VarInt(entity.GetType()),
		Pos:      proto.NewCoordinate(entity.GetPos()),
		Motion:   proto.LpVec3{X: motion[0], Y: motion[1], Z: motion[2]},
		Pitch:    proto.DegreesToAngle(rot[1]),
		Yaw:      yaw,
		HeadYaw:  yaw,
	}
}

type TeleportEntity struct {
	EntityID proto.VarInt
	Position proto.Coordinate
	Velocity proto.Coordinate
	Yaw      proto.Float
	Pitch    proto.Float
	OnGround proto.Boolean
}

func (t *TeleportEntity) WriteTo(w io.Writer) (n int64, err error) {
	fields := [6]io.WriterTo{
		t.EntityID,
		t.Position, t.Velocity,
		t.Yaw, t.Pitch,
		t.OnGround,
	}
	for _, f := range fields {
		nn, err := f.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

func NewTeleportEntity(entityID int32, pos [3]float64, rot [2]float32, onGround bool) *TeleportEntity {
	// todo: create a helper for float angle conversion
	return &TeleportEntity{
		EntityID: proto.VarInt(entityID),
		Position: proto.NewCoordinate(pos),
		Yaw:      proto.Float(rot[0] * 256 / 360),
		Pitch:    proto.Float(rot[1] * 256 / 360),
		OnGround: proto.Boolean(onGround),
	}
}

type Login struct {
	EntityID            proto.Int
	IsHardcore          proto.Boolean
	DimensionNames      proto.PrefixedArray[proto.Identifier, *proto.Identifier]
	MaxPlayers          proto.VarInt
	ViewDistance        proto.VarInt
	SimulationDistance  proto.VarInt
	ReducedDebugInfo    proto.Boolean
	EnableRespawnScreen proto.Boolean
	DoLimitedCrafting   proto.Boolean
	DimensionType       proto.VarInt
	DimensionName       proto.Identifier
	HashedSeed          proto.Long
	GameMode            proto.UnsignedByte
	PreviousGameMode    proto.Byte
	IsDebug             proto.Boolean
	IsFlat              proto.Boolean
	HasDeathLocation    proto.Boolean
	PortalCooldown      proto.VarInt
	SeaLevel            proto.VarInt
	EnforceSecureChat   proto.Boolean
}

func (l *Login) WriteTo(w io.Writer) (n int64, err error) {
	fields := [20]io.WriterTo{
		l.EntityID, l.IsHardcore, l.DimensionNames,
		l.MaxPlayers, l.ViewDistance, l.SimulationDistance,
		l.ReducedDebugInfo, l.EnableRespawnScreen, l.DoLimitedCrafting,
		l.DimensionType, l.DimensionName,
		l.HashedSeed,
		l.GameMode, l.PreviousGameMode,
		l.IsDebug, l.IsFlat, l.HasDeathLocation,
		l.PortalCooldown, l.SeaLevel,
		l.EnforceSecureChat,
	}
	for _, f := range fields {
		nn, err := f.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

type Respawn struct {
	DimensionType    proto.VarInt
	DimensionName    proto.Identifier
	HashedSeed       proto.Long
	GameMode         proto.UnsignedByte
	PreviousGameMode proto.Byte
	IsDebug          proto.Boolean
	IsFlat           proto.Boolean
	HasDeathLocation proto.Boolean
	PortalCooldown   proto.VarInt
	SeaLevel         proto.VarInt
	DataKept         proto.Byte
}

func (r *Respawn) WriteTo(w io.Writer) (n int64, err error) {
	fields := [11]io.WriterTo{
		r.DimensionType, r.DimensionName,
		r.HashedSeed,
		r.GameMode, r.PreviousGameMode,
		r.IsDebug, r.IsFlat, r.HasDeathLocation,
		r.PortalCooldown, r.SeaLevel,
		r.DataKept,
	}
	for _, f := range fields {
		nn, err := f.WriteTo(w)
		n += nn
		if err != nil {
			return n, err
		}
	}
	return n, nil
}
