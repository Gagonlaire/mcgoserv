package server

import (
	"io"
	"math/rand"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc/block"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func (c *Connection) applyPlaceResult(primary world.BlockPos, result world.PlaceResult) {
	dim := c.Server.World.GetEntityDimension(c.Player)
	if !result.OK {
		state, _ := dim.GetState(primary)
		c.broadcastBlockUpdate(dim, primary, state)
		return
	}
	primaryHasWrite := false
	for _, w := range result.Writes {
		c.broadcastBlockUpdate(dim, w.Pos, w.NewState)
		if w.Pos == primary {
			primaryHasWrite = true
		}
	}
	if !primaryHasWrite {
		state, _ := dim.GetState(primary)
		c.broadcastBlockUpdate(dim, primary, state)
	}
	soundPos, soundState := representativeWrite(primary, result.Writes)
	c.broadcastPlaceSound(dim, soundPos, soundState)

	if result.OpenSignEdit != nil {
		c.sendOpenSignEditor(*result.OpenSignEdit, true)
	}
}

func (c *Connection) applyInteractResult(result world.PlaceResult) {
	if !result.OK {
		return
	}
	if result.OpenSignEdit != nil {
		c.sendOpenSignEditor(*result.OpenSignEdit, true)
	}
}

func (c *Connection) sendOpenSignEditor(pos world.BlockPos, front bool) {
	// TODO(packet 1.21.10)
	pkt := c.NewPacket(
		packet.PlayClientboundOpenSignEditor,
		toProtoPosition(pos),
		proto.Boolean(front),
	)
	c.Send(pkt)
}

func (c *Connection) broadcastBlockEntityData(dim *world.Dimension, be world.Networked) {
	data, ok := be.NetworkData().(io.WriterTo)
	if !ok {
		return
	}
	pos := be.Pos()
	// TODO(packet 1.21.10)
	pkt := c.NewPacket(
		packet.PlayClientboundBlockEntityData,
		toProtoPosition(pos),
		proto.VarInt(int(be.Type())),
		data,
	)
	c.Server.BroadcastChunkWatchers(dim, pos.X>>4, pos.Z>>4, pkt)
}

func (c *Connection) applyBreakResult(primary world.BlockPos, result world.BreakResult) {
	if !result.OK {
		return
	}
	dim := c.Server.World.GetEntityDimension(c.Player)
	var primaryOldState int32
	for _, ch := range result.Changes {
		c.broadcastBlockUpdate(dim, ch.Pos, ch.NewState)
		if ch.Pos == primary {
			primaryOldState = ch.OldState
		}
	}

	// TODO: batch the BlockUpdates above into a SectionBlocksUpdate when ≥2 writes land in the same chunk section
	c.broadcastBreakEvent(dim, primary, primaryOldState)
}

func (c *Connection) broadcastBlockUpdate(dim *world.Dimension, pos world.BlockPos, state int32) {
	// TODO(packet 1.21.10)
	pkt := c.NewPacket(
		packet.PlayClientboundBlockUpdate,
		toProtoPosition(pos),
		proto.VarInt(int(state)),
	)
	c.Server.BroadcastChunkWatchers(dim, pos.X>>4, pos.Z>>4, pkt)
}

func (c *Connection) broadcastPlaceSound(dim *world.Dimension, pos world.BlockPos, state int32) {
	id, ok := block.FromStateID(int(state))
	if !ok {
		return
	}
	soundID, ok := id.SoundGroup().Place()
	if !ok {
		return
	}
	// TODO: pitch range/volume, verify vanilla constants
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pitch := 0.5 + r.Float64()*1.5
	// TODO(packet 1.21.10)
	pkt := c.NewPacket(
		packet.PlayClientboundSound,
		proto.VarInt(int(soundID)+1),
		proto.VarInt(4),
		proto.Int(pos.X*8),
		proto.Int(pos.Y*8),
		proto.Int(pos.Z*8),
		proto.Float(1),
		proto.Float(pitch),
		proto.Long(0),
	)
	c.Server.BroadcastChunkWatchersOthers(c, dim, pos.X>>4, pos.Z>>4, pkt)
}

func (c *Connection) broadcastBreakEvent(dim *world.Dimension, pos world.BlockPos, oldState int32) {
	// TODO(packet 1.21.10): replace hard coded event with helper for events
	pkt := c.NewPacket(
		packet.PlayClientboundLevelEvent,
		proto.Int(2001),
		toProtoPosition(pos),
		proto.Int(int(oldState)),
		proto.Boolean(false),
	)
	c.Server.BroadcastChunkWatchersOthers(c, dim, pos.X>>4, pos.Z>>4, pkt)
}

func representativeWrite(primary world.BlockPos, writes []world.BlockChange) (world.BlockPos, int32) {
	for _, w := range writes {
		if w.Pos == primary {
			return w.Pos, w.NewState
		}
	}
	if len(writes) == 0 {
		return primary, 0
	}
	return writes[0].Pos, writes[0].NewState
}

func toProtoPosition(p world.BlockPos) proto.Position {
	return proto.Position{X: int32(p.X), Y: int32(p.Y), Z: int32(p.Z)}
}
