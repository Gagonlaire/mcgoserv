package server

import (
	"log"
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
)

const (
	fixedPointMultiplier = 4096.0
	maxDelta             = 32767
	minDelta             = -32768
)

func (c *Connection) Teleport(x, y, z mc.Double, yaw, pitch mc.Float) {
	// todo: correct usage of tp id and flags, also add velocity
	pkt, _ := packet.NewPacket(packet.PlayClientboundSynchronizePlayerPosition,
		x, y, z,
		yaw, pitch,
		mc.Byte(0),
		mc.VarInt(0),
	)
	c.Send(pkt)
}

func (c *Connection) HandleConfirmTeleportationPacket(pkt *packet.Packet) {
	var teleportId mc.VarInt

	if err := pkt.Decode(&teleportId); err != nil {
		return
	}
}

func (c *Connection) HandleKeepAlivePacket(pkt *packet.Packet) {
	var keepAliveId mc.Long

	if err := pkt.Decode(&keepAliveId); err != nil {
		return
	}
	c.LastKeepAliveID = int64(keepAliveId)
	c.LastKeepAlive = c.server.World.Time
}

func (c *Connection) HandleClientTickEnd(_ *packet.Packet) {
	// Used for some specific logic
}

func (c *Connection) HandleMovePlayerPos(pkt *packet.Packet) {
	var x, y, z mc.Double
	var onGround mc.Boolean

	if err := pkt.Decode(&x, &y, &z, &onGround); err != nil {
		log.Printf("Error decoding move player pos packet: %v", err)
		return
	}
	oldX, oldY, oldZ := c.Player.Position.X, c.Player.Position.Y, c.Player.Position.Z
	if c.handlePositionUpdate(float64(x), float64(y), float64(z)) {
		c.syncMovement(float64(oldX), float64(oldY), float64(oldZ), true, false)
	}
}

func (c *Connection) HandleMovePlayerPosRot(pkt *packet.Packet) {
	var x, y, z mc.Double
	var yaw, pitch mc.Float
	var onGround mc.Boolean

	if err := pkt.Decode(&x, &y, &z, &yaw, &pitch, &onGround); err != nil {
		log.Printf("Error decoding move player pos rot packet: %v", err)
		return
	}

	oldX, oldY, oldZ := c.Player.Position.X, c.Player.Position.Y, c.Player.Position.Z
	posValid := c.handlePositionUpdate(float64(x), float64(y), float64(z))
	rotValid := c.handleRotationUpdate(float32(yaw), float32(pitch))

	if posValid || rotValid {
		c.syncMovement(float64(oldX), float64(oldY), float64(oldZ), posValid, rotValid)
	}
}

func (c *Connection) HandleMovePlayerRot(pkt *packet.Packet) {
	var yaw, pitch mc.Float
	var onGround mc.Boolean

	if err := pkt.Decode(&yaw, &pitch, &onGround); err != nil {
		log.Printf("Error decoding move player rot: %v", err)
		return
	}

	if c.handleRotationUpdate(float32(yaw), float32(pitch)) {
		c.syncMovement(float64(c.Player.Position.X), float64(c.Player.Position.Y), float64(c.Player.Position.Z), false, true)
	}
}

func (c *Connection) handlePositionUpdate(x, y, z float64) bool {
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
		c.Teleport(c.Player.Position.X, c.Player.Position.Y, c.Player.Position.Z, c.Player.Position.Yaw, c.Player.Position.Pitch)
		return false
	}

	c.Player.Position.X = mc.Double(x)
	c.Player.Position.Y = mc.Double(y)
	c.Player.Position.Z = mc.Double(z)

	return true
}

func (c *Connection) syncMovement(oldX, oldY, oldZ float64, posChanged, rotChanged bool) {
	deltaX := int64(float64(c.Player.Position.X)*fixedPointMultiplier - oldX*fixedPointMultiplier)
	deltaY := int64(float64(c.Player.Position.Y)*fixedPointMultiplier - oldY*fixedPointMultiplier)
	deltaZ := int64(float64(c.Player.Position.Z)*fixedPointMultiplier - oldZ*fixedPointMultiplier)
	needsTeleport := deltaX > maxDelta || deltaX < minDelta ||
		deltaY > maxDelta || deltaY < minDelta ||
		deltaZ > maxDelta || deltaZ < minDelta

	if needsTeleport {
		c.broadcastTeleport()
		return
	}

	yaw := mc.Angle(c.Player.Position.Yaw / 360.0 * 256.0)
	pitch := mc.Angle(c.Player.Position.Pitch / 360.0 * 256.0)
	var pkt *packet.Packet

	switch {
	case posChanged && rotChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundUpdateEntityPositionAndRot,
			c.Player.EntityID,
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			yaw, pitch,
			mc.Boolean(c.Player.Position.Flags == 1),
		)
	case posChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundUpdateEntityPosition,
			c.Player.EntityID,
			mc.Short(deltaX), mc.Short(deltaY), mc.Short(deltaZ),
			mc.Boolean(c.Player.Position.Flags == 1),
		)
	case rotChanged:
		pkt, _ = packet.NewPacket(packet.PlayClientboundUpdateEntityRotation,
			c.Player.EntityID,
			yaw, pitch,
			mc.Boolean(c.Player.Position.Flags == 1),
		)
	}

	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))

	if rotChanged {
		pktHead, _ := packet.NewPacket(packet.PlayClientboundRotateHead,
			c.Player.EntityID,
			yaw,
		)
		c.server.Broadcaster.Broadcast(pktHead, systems.NotSender(c))
	}
}

func (c *Connection) handleRotationUpdate(yaw, pitch float32) bool {
	if math.IsNaN(float64(yaw)) || math.IsNaN(float64(pitch)) ||
		math.IsInf(float64(yaw), 0) || math.IsInf(float64(pitch), 0) {
		// todo: change to a disconnect method with a reason
		c.close()
		return false
	}

	c.Player.Position.Yaw = mc.Float(yaw)
	c.Player.Position.Pitch = mc.Float(pitch)
	return true
}

func (c *Connection) SendKeepAlive() {
	var packetId int

	if c.State == mc.StateConfiguration {
		packetId = packet.ConfigurationClientboundKeepAlive
	} else if c.State == mc.StatePlay {
		packetId = packet.PlayClientboundKeepAlive
	} else {
		panic("Invalid state for sending keep-alive")
	}

	c.LastKeepAliveID = c.server.World.Time
	pkt, _ := packet.NewPacket(packetId, mc.Long(c.server.World.Time))
	c.Send(pkt)
}

func (c *Connection) broadcastTeleport() {
	pkt, _ := packet.NewPacket(packet.PlayClientboundTeleportEntity,
		c.Player.EntityID,
		c.Player.Position.X, c.Player.Position.Y, c.Player.Position.Z,
		mc.Double(0), mc.Double(0), mc.Double(0),
		c.Player.Position.Yaw*256/360, c.Player.Position.Pitch*256/360,
		c.Player.Position.Flags,
	)
	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}

func (c *Connection) HandlePlayerCommand(pkt *packet.Packet) {
	var eID mc.VarInt
	var actionID mc.VarInt
	var jumpBoost mc.VarInt

	if err := pkt.Decode(&eID, &actionID, &jumpBoost); err != nil {
		log.Printf("Error decoding player command packet: %v", err)
	}

	switch mc.PlayerCommand(actionID) {
	case mc.ActionStartSprinting:
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			c.Player.EntityID,
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0x08),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	case mc.ActionStopSprinting:
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			c.Player.EntityID,
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	}
}

func (c *Connection) HandlePlayerInput(pkt *packet.Packet) {
	var input mc.UnsignedByte

	if err := pkt.Decode(&input); err != nil {
		log.Printf("Error decoding player input packet: %v", err)
	}

	c.Player.Input = input

	if input&mc.InputSneak != 0 {
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			c.Player.EntityID,
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0x08),
			mc.UnsignedByte(6),
			mc.VarInt(20),
			mc.VarInt(mc.PoseSneaking),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	} else {
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			c.Player.EntityID,
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0),
			mc.UnsignedByte(6),
			mc.VarInt(20),
			mc.VarInt(mc.PoseStanding),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	}
}

func (c *Connection) HandlePlayerLoaded(_ *packet.Packet) {
	c.Player.Loaded = true
}

func (c *Connection) HandleMovePlayerStatusOnly(pkt *packet.Packet) {
	var newFlags mc.Byte

	if err := pkt.Decode(&newFlags); err != nil {
		log.Printf("Error decoding move player status only packet: %v", err)
		return
	}
	c.Player.Position.Flags = newFlags
	// we use entity rotation because this is the smallest packer that can send flags
	c.syncMovement(
		float64(c.Player.Position.X),
		float64(c.Player.Position.Y),
		float64(c.Player.Position.Z),
		false,
		true,
	)
}
