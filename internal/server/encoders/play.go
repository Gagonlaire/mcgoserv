package encoders

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
)

type AddEntity struct {
	EntityID mc.VarInt
	UUID     mc.UUID
	TypeID   mc.VarInt
	Pos      mc.Coordinate
	Motion   mc.LpVec3
	Pitch    mc.Angle
	Yaw      mc.Angle
	HeadYaw  mc.Angle
	Data     mc.VarInt
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

func NewAddEntity(entity *entities.BaseEntity) *AddEntity {
	yaw := mc.DegreesToAngle(entity.Rot[0])
	return &AddEntity{
		EntityID: mc.VarInt(entity.EntityID),
		UUID:     mc.UUID(entity.UUID),
		TypeID:   mc.VarInt(entity.TypeID),
		Pos:      mc.NewCoordinate(entity.Pos),
		Motion:   mc.LpVec3{X: entity.Motion[0], Y: entity.Motion[1], Z: entity.Motion[2]},
		Pitch:    mc.DegreesToAngle(entity.Rot[1]),
		Yaw:      yaw,
		HeadYaw:  yaw,
	}
}

type TeleportEntity struct {
	EntityID mc.VarInt
	Pos      mc.Coordinate
	Velocity mc.Coordinate
	Yaw      mc.Float
	Pitch    mc.Float
	OnGround mc.Boolean
}

func (t *TeleportEntity) WriteTo(w io.Writer) (n int64, err error) {
	fields := [6]io.WriterTo{
		t.EntityID,
		t.Pos, t.Velocity,
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
		EntityID: mc.VarInt(entityID),
		Pos:      mc.NewCoordinate(pos),
		Yaw:      mc.Float(rot[0] * 256 / 360),
		Pitch:    mc.Float(rot[1] * 256 / 360),
		OnGround: mc.Boolean(onGround),
	}
}

type Login struct {
	EntityID            mc.Int
	IsHardcore          mc.Boolean
	DimensionNames      mc.PrefixedArray[mc.Identifier, *mc.Identifier]
	MaxPlayers          mc.VarInt
	ViewDistance        mc.VarInt
	SimulationDistance  mc.VarInt
	ReducedDebugInfo    mc.Boolean
	EnableRespawnScreen mc.Boolean
	DoLimitedCrafting   mc.Boolean
	DimensionType       mc.VarInt
	DimensionName       mc.Identifier
	HashedSeed          mc.Long
	GameMode            mc.UnsignedByte
	PreviousGameMode    mc.Byte
	IsDebug             mc.Boolean
	IsFlat              mc.Boolean
	HasDeathLocation    mc.Boolean
	PortalCooldown      mc.VarInt
	SeaLevel            mc.VarInt
	EnforceSecureChat   mc.Boolean
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
