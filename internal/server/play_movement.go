package server

import (
	"log"
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
)

const (
	fixedPointMultiplier = 4096.0
	maxDelta             = 32767
	minDelta             = -32768
)

func (c *Connection) HandleConfirmTeleportation(teleportID *mc.VarInt) {
	// todo: unlock movement packets
}

func (c *Connection) HandleSetPlayerPosition(data *decoders.SetPlayerPosition) {
	oldX, oldY, oldZ := c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2]
	if c.handlePositionUpdate(float64(data.X), float64(data.Y), float64(data.Z), int8(data.Flags)) {
		c.syncMovement(oldX, oldY, oldZ, true, false)
	}
}

func (c *Connection) HandleSetPlayerPositionAndRotation(data *decoders.SetPlayerPositionAndRotation) {
	oldX, oldY, oldZ := c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2]
	posValid := c.handlePositionUpdate(float64(data.X), float64(data.Y), float64(data.Z), int8(data.Flags))
	rotValid := c.handleRotationUpdate(float32(data.Yaw), float32(data.Pitch), int8(data.Flags))

	if posValid || rotValid {
		c.syncMovement(oldX, oldY, oldZ, posValid, rotValid)
	}
}

func (c *Connection) HandleSetPlayerRotation(data *decoders.SetPlayerRotation) {
	if c.handleRotationUpdate(float32(data.Yaw), float32(data.Pitch), int8(data.Flags)) {
		c.syncMovement(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], false, true)
	}
}

func (c *Connection) HandleSetPlayerMovementFlags(flags *mc.Byte) {
	c.Player.OnGround = (*flags)&0x01 != 0
	c.syncMovement(
		c.Player.Pos[0],
		c.Player.Pos[1],
		c.Player.Pos[2],
		false,
		true,
	)
}

func (c *Connection) handleRotationUpdate(yaw, pitch float32, flags int8) bool {
	if math.IsNaN(float64(yaw)) || math.IsNaN(float64(pitch)) ||
		math.IsInf(float64(yaw), 0) || math.IsInf(float64(pitch), 0) {
		// todo: change to a disconnect method with a reason
		c.close()
		return false
	}

	c.Player.Rot[0] = yaw
	c.Player.Rot[1] = pitch
	c.Player.OnGround = flags&0x01 != 0
	c.Player.PushingAgainstWall = flags&0x02 != 0

	return true
}

func (c *Connection) syncMovement(oldX, oldY, oldZ float64, posChanged, rotChanged bool) {
	deltaX := int64(c.Player.Pos[0]*fixedPointMultiplier - oldX*fixedPointMultiplier)
	deltaY := int64(c.Player.Pos[1]*fixedPointMultiplier - oldY*fixedPointMultiplier)
	deltaZ := int64(c.Player.Pos[2]*fixedPointMultiplier - oldZ*fixedPointMultiplier)
	needsTeleport := deltaX > maxDelta || deltaX < minDelta ||
		deltaY > maxDelta || deltaY < minDelta ||
		deltaZ > maxDelta || deltaZ < minDelta

	if needsTeleport {
		c.broadcastTeleport()
		return
	}

	yaw := mc.Angle(c.Player.Rot[0] / 360.0 * 256.0)
	pitch := mc.Angle(c.Player.Rot[1] / 360.0 * 256.0)
	var pkt *packet.Packet

	switch {
	case posChanged && rotChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundMoveEntityPosRot,
			mc.VarInt(c.Player.EntityID),
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			yaw, pitch,
			mc.Boolean(c.Player.OnGround),
		)
	case posChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundMoveEntityPos,
			mc.VarInt(c.Player.EntityID),
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			mc.Boolean(c.Player.OnGround),
		)
	case rotChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundMoveEntityRot,
			mc.VarInt(c.Player.EntityID),
			yaw, pitch,
			mc.Boolean(c.Player.OnGround),
		)
	}

	c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))

	if rotChanged {
		pktHead, _ := packet.NewPacket(packet.PlayClientboundRotateHead,
			mc.VarInt(c.Player.EntityID),
			yaw,
		)
		c.Server.Broadcaster.Broadcast(pktHead, systems.NotSender(c))
	}
}

func (c *Connection) handlePositionUpdate(x, y, z float64, flags int8) bool {
	// todo: implement logic for vehicule, sleeping and flying.
	// todo: ignore movement packets if there is a teleportation pending

	if math.IsNaN(x) || math.IsNaN(y) || math.IsNaN(z) ||
		math.IsInf(x, 0) || math.IsInf(y, 0) || math.IsInf(z, 0) {
		// todo: disconnect for invalid movement
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
		log.Printf("%s moved too quickly! %.2f, %.2f, %.2f", c.Player.Name, dx, dy, dz)
		c.teleport(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], c.Player.Rot[0], c.Player.Rot[1])
		return false
	}

	c.Player.Pos[0] = x
	c.Player.Pos[1] = y
	c.Player.Pos[2] = z
	c.Player.OnGround = flags&0x01 != 0
	c.Player.PushingAgainstWall = flags&0x02 != 0
	c.Server.World.UpdateEntityChunk(c.Player.EntityID, c.Player.Movement.LastTickX, c.Player.Movement.LastTickZ, x, z)
	c.updateChunkView(false)

	return true
}

func (c *Connection) broadcastTeleport() {
	pkt, _ := packet.NewPacket(packet.PlayClientboundTeleportEntity,
		mc.VarInt(c.Player.EntityID),
		mc.Double(c.Player.Pos[0]), mc.Double(c.Player.Pos[1]), mc.Double(c.Player.Pos[2]),
		mc.Double(0), mc.Double(0), mc.Double(0),
		mc.Float(c.Player.Rot[0]*256/360), mc.Float(c.Player.Rot[1]*256/360),
		mc.Boolean(c.Player.OnGround),
	)
	c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}

func (c *Connection) teleport(x, y, z float64, yaw, pitch float32) {
	// todo: correct usage of tp id and flags, also add velocity
	pkt, _ := packet.NewPacket(
		packet.PlayClientboundPlayerPosition,
		mc.Double(x), mc.Double(y), mc.Double(z),
		mc.Float(yaw), mc.Float(pitch),
		mc.Byte(0),
		mc.VarInt(0),
	)
	c.Send(pkt)
}

func (c *Connection) updateChunkView(force bool) {
	// todo: check chunk batch start/stop
	cx, cz := world.GetChunkPosition(c.Player.Pos[0], c.Player.Pos[2])

	if cx == c.Player.Movement.LastChunkX && cz == c.Player.Movement.LastChunkZ && !force {
		return
	}

	dim := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
	loadRadius := int(c.Player.Information.ViewDistance) + 1
	keepChunks := make(map[mc.ChunkPos]bool, (loadRadius*2+1)*(loadRadius*2+1))
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			keepChunks[mc.ChunkPos{X: x, Z: z}] = true
		}
	}

	centerPkt, _ := packet.NewPacket(packet.PlayClientboundSetChunkCacheCenter, mc.VarInt(cx), mc.VarInt(cz))
	c.Send(centerPkt)
	selfID := c.Player.EntityID

	for pos := range c.Player.Movement.VisibleChunks {
		if keepChunks[pos] {
			continue
		}
		chunk := dim.GetChunk(pos.X, pos.Z)
		if count := len(chunk.Entities); count > 0 {
			removePkt, _ := packet.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(count))
			for entityID := range chunk.Entities {
				_ = removePkt.Encode(mc.VarInt(entityID))
			}
			c.Send(removePkt)
		}
		delete(chunk.Watchers, selfID)
		delete(c.Player.Movement.VisibleChunks, pos)
		forgetPkt, _ := packet.NewPacket(packet.PlayClientboundForgetLevelChunk, mc.Int(pos.Z), mc.Int(pos.X))
		c.Send(forgetPkt)
	}
	for x := cx - loadRadius; x <= cx+loadRadius; x++ {
		for z := cz - loadRadius; z <= cz+loadRadius; z++ {
			pos := mc.ChunkPos{X: x, Z: z}
			if _, known := c.Player.Movement.VisibleChunks[pos]; known {
				continue
			}
			chunk := dim.GetChunk(x, z)
			chunkPkt, _ := packet.NewPacket(packet.PlayClientboundLevelChunkWithLight, chunk)
			c.Send(chunkPkt)
			for entityID := range chunk.Entities {
				if entityID == selfID {
					continue
				}
				c.SendSpawnEntity(c.Server.World.EntitiesByID[entityID])
			}
			chunk.Watchers[selfID] = struct{}{}
			c.Player.Movement.VisibleChunks[pos] = struct{}{}
		}
	}

	oldChunk := dim.GetChunk(c.Player.Movement.LastChunkX, c.Player.Movement.LastChunkZ)
	newChunk := dim.GetChunk(cx, cz)
	selfEntity := &c.Player.LivingEntity.BaseEntity
	removePkt, _ := packet.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), mc.VarInt(selfID))

	for watcherID := range oldChunk.Watchers {
		if watcherID == selfID {
			continue
		}
		if _, stillSees := newChunk.Watchers[watcherID]; stillSees {
			continue
		}
		c.Server.ConnectionsByEID[watcherID].Send(removePkt)
	}
	for watcherID := range newChunk.Watchers {
		if watcherID == selfID {
			continue
		}
		if _, alreadySaw := oldChunk.Watchers[watcherID]; alreadySaw {
			continue
		}
		c.Server.ConnectionsByEID[watcherID].SendSpawnEntity(selfEntity)
	}

	c.Player.Movement.LastChunkX = cx
	c.Player.Movement.LastChunkZ = cz
}
