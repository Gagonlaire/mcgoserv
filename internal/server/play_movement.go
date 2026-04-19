package server

import (
	"math"
	"math/rand/v2"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
)

const (
	fixedPointMultiplier = 4096.0
	maxDelta             = 32767
	minDelta             = -32768
)

func (c *Connection) HandleConfirmTeleportation(teleportID *mc.VarInt) {
	if int32(*teleportID) == c.Player.Movement.PendingTeleport {
		c.Player.Movement.PendingTeleport = 0
	}
}

func (c *Connection) HandleSetPlayerPosition(data *decoders.SetPlayerPosition) {
	oldX, oldY, oldZ := c.Player.Position[0], c.Player.Position[1], c.Player.Position[2]
	if c.handlePositionUpdate(float64(data.X), float64(data.Y), float64(data.Z), int8(data.Flags)) {
		c.syncMovement(oldX, oldY, oldZ, true, false)
	}
}

func (c *Connection) HandleSetPlayerPositionAndRotation(data *decoders.SetPlayerPositionAndRotation) {
	oldX, oldY, oldZ := c.Player.Position[0], c.Player.Position[1], c.Player.Position[2]
	posValid := c.handlePositionUpdate(float64(data.X), float64(data.Y), float64(data.Z), int8(data.Flags))
	rotValid := c.handleRotationUpdate(float32(data.Yaw), float32(data.Pitch), int8(data.Flags))

	if posValid || rotValid {
		c.syncMovement(oldX, oldY, oldZ, posValid, rotValid)
	}
}

func (c *Connection) HandleSetPlayerRotation(data *decoders.SetPlayerRotation) {
	if c.handleRotationUpdate(float32(data.Yaw), float32(data.Pitch), int8(data.Flags)) {
		c.syncMovement(c.Player.Position[0], c.Player.Position[1], c.Player.Position[2], false, true)
	}
}

func (c *Connection) HandleSetPlayerMovementFlags(flags *mc.Byte) {
	if c.Player.Movement.PendingTeleport != 0 {
		return
	}
	c.Player.OnGround = (*flags)&0x01 != 0
	c.syncMovement(
		c.Player.Position[0],
		c.Player.Position[1],
		c.Player.Position[2],
		false,
		true,
	)
}

func (c *Connection) handleRotationUpdate(yaw, pitch float32, flags int8) bool {
	if c.Player.Movement.PendingTeleport != 0 {
		return false
	}

	if math.IsNaN(float64(yaw)) || math.IsNaN(float64(pitch)) ||
		math.IsInf(float64(yaw), 0) || math.IsInf(float64(pitch), 0) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPlayerMovement))
		return false
	}

	c.Player.Rotation[0] = yaw
	c.Player.Rotation[1] = pitch
	c.Player.OnGround = flags&0x01 != 0
	c.Player.PushingAgainstWall = flags&0x02 != 0

	return true
}

func (c *Connection) syncMovement(oldX, oldY, oldZ float64, posChanged, rotChanged bool) {
	deltaX := int64(c.Player.Position[0]*fixedPointMultiplier - oldX*fixedPointMultiplier)
	deltaY := int64(c.Player.Position[1]*fixedPointMultiplier - oldY*fixedPointMultiplier)
	deltaZ := int64(c.Player.Position[2]*fixedPointMultiplier - oldZ*fixedPointMultiplier)
	needsTeleport := deltaX > maxDelta || deltaX < minDelta ||
		deltaY > maxDelta || deltaY < minDelta ||
		deltaZ > maxDelta || deltaZ < minDelta

	if needsTeleport {
		if logger.IsDebug() {
			logger.Debug("Delta too large for %s, using teleport sync", c.Player.Name)
		}
		c.Teleport(c.Player.Position[0], c.Player.Position[1], c.Player.Position[2], c.Player.Rotation[0], c.Player.Rotation[1], 0)
		return
	}

	yaw := mc.DegreesToAngle(c.Player.Rotation[0])
	pitch := mc.DegreesToAngle(c.Player.Rotation[1])
	var pkt *packet.OutboundPacket

	switch {
	case posChanged && rotChanged:
		pkt = c.NewPacket(packet.PlayClientboundMoveEntityPosRot,
			mc.VarInt(c.Player.EntityID),
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			yaw, pitch,
			mc.Boolean(c.Player.OnGround),
		)
	case posChanged:
		pkt = c.NewPacket(packet.PlayClientboundMoveEntityPos,
			mc.VarInt(c.Player.EntityID),
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			mc.Boolean(c.Player.OnGround),
		)
	case rotChanged:
		pkt = c.NewPacket(packet.PlayClientboundMoveEntityRot,
			mc.VarInt(c.Player.EntityID),
			yaw, pitch,
			mc.Boolean(c.Player.OnGround),
		)
	}

	c.Server.BroadcastViewers(c, pkt)

	if rotChanged {
		pktHead := c.NewPacket(packet.PlayClientboundRotateHead,
			mc.VarInt(c.Player.EntityID),
			yaw,
		)
		c.Server.BroadcastViewers(c, pktHead)
	}
}

func (c *Connection) handlePositionUpdate(x, y, z float64, flags int8) bool {
	// todo: implement logic for vehicle, sleeping and flying.
	if c.Player.Movement.PendingTeleport != 0 {
		return false
	}

	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPlayerMovement))
		return false
	}

	x = math.Max(-30000000, math.Min(30000000, x))
	z = math.Max(-30000000, math.Min(30000000, z))
	y = math.Max(-20000000, math.Min(20000000, y))

	dx := x - c.Player.Movement.LastTickX
	dy := y - c.Player.Movement.LastTickY
	dz := z - c.Player.Movement.LastTickZ
	distSq := dx*dx + dy*dy + dz*dz

	velocitySq := 0.0

	c.Player.Movement.PacketCount++
	multiplier := c.Player.Movement.PacketCount
	if c.Player.Movement.PacketCount > 5 {
		multiplier = 1
	}

	maxDistSq := 100.0 * float64(multiplier)
	if distSq-velocitySq > maxDistSq {
		logger.Warn("%s moved too quickly! %.2f, %.2f, %.2f", c.Player.Name, dx, dy, dz)
		c.Teleport(c.Player.Position[0], c.Player.Position[1], c.Player.Position[2], c.Player.Rotation[0], c.Player.Rotation[1], 0)
		return false
	}

	c.Player.Position[0] = x
	c.Player.Position[1] = y
	c.Player.Position[2] = z
	c.Player.OnGround = flags&0x01 != 0
	c.Player.PushingAgainstWall = flags&0x02 != 0
	c.Server.World.UpdateEntityChunk(c.Player.EntityID, c.Player.Movement.LastTickX, c.Player.Movement.LastTickZ, x, z)
	c.updateChunkView(false)

	return true
}

func (c *Connection) Teleport(x, y, z float64, yaw, pitch float32, flags mc.TeleportationFlags) {
	if logger.IsDebug() {
		logger.Debug("Teleporting %s to %.2f, %.2f, %.2f", c.Player.Name, x, y, z)
	}
	oldX, oldZ := c.Player.Position[0], c.Player.Position[2]

	// todo: move this logic into a helper func
	if flags&mc.TeleportationFlagsRelativeX != 0 {
		x += c.Player.Position[0]
	}
	if flags&mc.TeleportationFlagsRelativeY != 0 {
		y += c.Player.Position[1]
	}
	if flags&mc.TeleportationFlagsRelativeZ != 0 {
		z += c.Player.Position[2]
	}
	if flags&mc.TeleportationFlagsRelativeYaw != 0 {
		yaw += c.Player.Rotation[0]
	}
	if flags&mc.TeleportationFlagsRelativePitch != 0 {
		pitch += c.Player.Rotation[1]
	}

	c.Player.Position[0] = x
	c.Player.Position[1] = y
	c.Player.Position[2] = z
	c.Player.Rotation[0] = yaw
	c.Player.Rotation[1] = pitch
	c.Player.Movement.LastTickX = x
	c.Player.Movement.LastTickY = y
	c.Player.Movement.LastTickZ = z

	teleportID := rand.Int32N(math.MaxInt32)
	if teleportID == 0 {
		teleportID = 1
	}
	c.Player.Movement.PendingTeleport = teleportID
	pkt := c.NewPacket(
		packet.PlayClientboundPlayerPosition,
		mc.VarInt(teleportID),
		mc.Double(x), mc.Double(y), mc.Double(z),
		mc.Double(0), mc.Double(0), mc.Double(0),
		mc.Float(yaw), mc.Float(pitch),
		mc.Int(flags),
	)
	c.Send(pkt)
	c.Server.World.UpdateEntityChunk(c.Player.EntityID, oldX, oldZ, x, z)

	dim := c.Server.World.GetEntityDimension(c.Player)
	oldCX, oldCZ := world.GetChunkPosition(oldX, oldZ)
	newCX, newCZ := world.GetChunkPosition(x, z)
	oldChunk := dim.GetChunk(oldCX, oldCZ)
	newChunk := dim.GetChunk(newCX, newCZ)

	selfID := c.Player.EntityID
	selfEntity := c.Player

	for watcherID := range oldChunk.Watchers {
		if watcherID == selfID {
			continue
		}
		conn, ok := c.Server.ConnectionsByEID.Load(watcherID)
		if !ok {
			continue
		}
		target := conn.(*Connection)

		if _, seesAfter := newChunk.Watchers[watcherID]; seesAfter {
			tpPkt := c.NewPacket(
				packet.PlayClientboundEntityPositionSync,
				encoders.NewTeleportEntity(selfID, c.Player.Position, c.Player.Rotation, c.Player.OnGround),
			)
			target.Send(tpPkt)
		} else {
			if logger.IsDebug() {
				logger.Debug("%s left %s's view (teleport)", c.Player.Name, target.Player.Name)
			}
			removePkt := c.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), mc.VarInt(selfID))
			target.Send(removePkt)
		}
	}

	for watcherID := range newChunk.Watchers {
		if watcherID == selfID {
			continue
		}
		if _, sawBefore := oldChunk.Watchers[watcherID]; sawBefore {
			continue
		}
		if conn, ok := c.Server.ConnectionsByEID.Load(watcherID); ok {
			target := conn.(*Connection)
			if logger.IsDebug() {
				logger.Debug("%s entered %s's view (teleport)", c.Player.Name, target.Player.Name)
			}
			target.SendSpawnEntity(selfEntity)
		}
	}
	// todo: this create a double check on entity tracking, updateChunkView is doing too much,
	// the entity tracking should be in a separate system or a func
	c.updateChunkView(false)
}

// todo: doing too much, remove entity tracking and update
func (c *Connection) updateChunkView(force bool) {
	// todo: check chunk batch start/stop
	cx, cz := world.GetChunkPosition(c.Player.Position[0], c.Player.Position[2])

	if cx == c.Player.Movement.LastChunkX && cz == c.Player.Movement.LastChunkZ && !force {
		return
	}

	dim := c.Server.World.GetEntityDimension(c.Player)
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	keepChunks := c.Player.Movement.KeepChunks
	if keepChunks == nil {
		keepChunks = make(map[mc.ChunkPos]bool, (loadRadius*2+1)*(loadRadius*2+1))
		c.Player.Movement.KeepChunks = keepChunks
	} else {
		clear(keepChunks)
	}
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			keepChunks[mc.ChunkPos{X: x, Z: z}] = true
		}
	}

	centerPkt := c.NewPacket(packet.PlayClientboundSetChunkCacheCenter, mc.VarInt(cx), mc.VarInt(cz))
	c.Send(centerPkt)
	selfID := c.Player.EntityID

	for pos := range c.Player.Movement.VisibleChunks {
		if keepChunks[pos] {
			continue
		}
		chunk := dim.GetChunk(pos.X, pos.Z)
		if count := len(chunk.Entities); count > 0 {
			removePkt := c.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(count))
			if removePkt != nil {
				for entityID := range chunk.Entities {
					_ = removePkt.Encode(mc.VarInt(entityID))
				}
				c.Send(removePkt)
			}
		}
		delete(chunk.Watchers, selfID)
		delete(c.Player.Movement.VisibleChunks, pos)
		forgetPkt := c.NewPacket(packet.PlayClientboundForgetLevelChunk, mc.Int(pos.Z), mc.Int(pos.X))
		c.Send(forgetPkt)
	}
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			pos := mc.ChunkPos{X: x, Z: z}
			if _, known := c.Player.Movement.VisibleChunks[pos]; known {
				continue
			}
			chunk := dim.GetChunk(x, z)
			chunkPkt := c.NewPacket(packet.PlayClientboundLevelChunkWithLight, chunk)
			c.Send(chunkPkt)
			c.SendChunkEntities(chunk)
			chunk.Watchers[selfID] = struct{}{}
			c.Player.Movement.VisibleChunks[pos] = struct{}{}
		}
	}

	oldChunk := dim.GetChunk(c.Player.Movement.LastChunkX, c.Player.Movement.LastChunkZ)
	newChunk := dim.GetChunk(cx, cz)
	selfEntity := c.Player
	removePkt := c.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), mc.VarInt(selfID))
	if removePkt != nil {
		for watcherID := range oldChunk.Watchers {
			if watcherID == selfID {
				continue
			}
			if _, stillSees := newChunk.Watchers[watcherID]; stillSees {
				continue
			}
			if conn, ok := c.Server.ConnectionsByEID.Load(watcherID); ok {
				removePkt.Retain()
				target := conn.(*Connection)
				select {
				case target.OutboundPackets <- removePkt:
				case <-target.ctx.Done():
					removePkt.Free()
				}
			}
		}
		removePkt.Free()
	}
	for watcherID := range newChunk.Watchers {
		if watcherID == selfID {
			continue
		}
		if _, alreadySaw := oldChunk.Watchers[watcherID]; alreadySaw {
			continue
		}
		if conn, ok := c.Server.ConnectionsByEID.Load(watcherID); ok {
			conn.(*Connection).SendSpawnEntity(selfEntity)
		}
	}

	c.Player.Movement.LastChunkX = cx
	c.Player.Movement.LastChunkZ = cz
}
