package server

import (
	"fmt"
	"log"
	"math"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Tnze/go-mc/nbt"
)

const (
	fixedPointMultiplier = 4096.0
	maxDelta             = 32767
	minDelta             = -32768
)

func (c *Connection) Teleport(x, y, z float64, yaw, pitch float32) {
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
	var flags mc.Byte

	if err := pkt.Decode(&x, &y, &z, &flags); err != nil {
		log.Printf("Error decoding move player pos packet: %v", err)
		return
	}
	oldX, oldY, oldZ := c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2]
	if c.handlePositionUpdate(float64(x), float64(y), float64(z), int8(flags)) {
		c.syncMovement(oldX, oldY, oldZ, true, false)
	}
}

func (c *Connection) HandleMovePlayerPosRot(pkt *packet.Packet) {
	var x, y, z mc.Double
	var yaw, pitch mc.Float
	var flags mc.Byte

	if err := pkt.Decode(&x, &y, &z, &yaw, &pitch, &flags); err != nil {
		log.Printf("Error decoding move player pos rot packet: %v", err)
		return
	}

	oldX, oldY, oldZ := c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2]
	posValid := c.handlePositionUpdate(float64(x), float64(y), float64(z), int8(flags))
	rotValid := c.handleRotationUpdate(float32(yaw), float32(pitch), int8(flags))

	if posValid || rotValid {
		c.syncMovement(oldX, oldY, oldZ, posValid, rotValid)
	}
}

func (c *Connection) HandleMovePlayerRot(pkt *packet.Packet) {
	var yaw, pitch mc.Float
	var flags mc.Byte

	if err := pkt.Decode(&yaw, &pitch, &flags); err != nil {
		log.Printf("Error decoding move player rot: %v", err)
		return
	}

	if c.handleRotationUpdate(float32(yaw), float32(pitch), int8(flags)) {
		c.syncMovement(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], false, true)
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
		c.Teleport(c.Player.Pos[0], c.Player.Pos[1], c.Player.Pos[2], c.Player.Rot[0], c.Player.Rot[1])
		return false
	}

	c.Player.Pos[0] = x
	c.Player.Pos[1] = y
	c.Player.Pos[2] = z
	c.Player.OnGround = flags&0x01 != 0
	c.Player.PushingAgainstWall = flags&0x02 != 0

	return true
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

	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))

	if rotChanged {
		pktHead, _ := packet.NewPacket(packet.PlayClientboundRotateHead,
			mc.VarInt(c.Player.EntityID),
			yaw,
		)
		c.server.Broadcaster.Broadcast(pktHead, systems.NotSender(c))
	}
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
		mc.VarInt(c.Player.EntityID),
		mc.Double(c.Player.Pos[0]), mc.Double(c.Player.Pos[1]), mc.Double(c.Player.Pos[2]),
		mc.Double(0), mc.Double(0), mc.Double(0),
		mc.Float(c.Player.Rot[0]*256/360), mc.Float(c.Player.Rot[1]*256/360),
		mc.Boolean(c.Player.OnGround),
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

	// todo: jumping seems to stop sprinting animation particles
	switch mc.PlayerCommand(actionID) {
	case mc.ActionStartSprinting:
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0x08),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	case mc.ActionStopSprinting:
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
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
			mc.VarInt(c.Player.EntityID),
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0x02),
			mc.UnsignedByte(6),
			mc.VarInt(20),
			mc.VarInt(mc.PoseSneaking),
			mc.UnsignedByte(0xff),
		)
		c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	} else {
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
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
	c.Player.OnGround = newFlags&0x01 != 0
	c.syncMovement(
		c.Player.Pos[0],
		c.Player.Pos[1],
		c.Player.Pos[2],
		false,
		true,
	)
}

func (c *Connection) HandleSwingArm(pkt *packet.Packet) {
	var arm mc.VarInt

	if err := pkt.Decode(&arm); err != nil {
		log.Printf("Error decoding swing arm packet: %v", err)
	}
	var animationID int

	if arm == 0 {
		animationID = 0
	} else {
		animationID = 3
	}

	c.AnimateEntity(animationID)
}

func (c *Connection) HandlePlayerAction(pkt *packet.Packet) {
	const (
		StatusStartDigging   = 0
		StatusCancelDigging  = 1
		StatusFinishDigging  = 2
		StatusDropItemStack  = 3
		StatusDropItem       = 4
		StatusReleaseUseItem = 5
		StatusSwapHand       = 6
	)

	var status mc.VarInt
	var location mc.Position
	var face mc.Byte
	var sequence mc.VarInt

	if err := pkt.Decode(&status, &location, &face, &sequence); err != nil {
		log.Printf("Error decoding player action packet: %v", err)
		return
	}

	switch status {
	case StatusStartDigging:
		if c.Player.GameMode == 1 {
			pkt, _ := packet.NewPacket(
				packet.PlayClientboundBlockUpdate,
				location,
				mc.VarInt(0),
			)
			eventPkt, _ := packet.NewPacket(
				packet.PlayClientboundLevelEvent,
				mc.Int(2001),
				location,
				mc.Int(3), // todo: this should check chunk to get the block state id
				mc.Boolean(false),
			)
			c.server.Broadcaster.Broadcast(eventPkt, systems.NotSender(c))
			c.server.Broadcaster.Broadcast(pkt)
		}
	case StatusFinishDigging:
		pkt, _ := packet.NewPacket(
			packet.PlayClientboundBlockUpdate,
			location,
			mc.VarInt(0),
		)
		c.server.Broadcaster.Broadcast(pkt)
	}

	pkt, _ = packet.NewPacket(packet.PlayClientboundBlockChangedAck, sequence)
	c.Send(pkt)
}

func (c *Connection) AnimateEntity(animationID int) {
	pkt, _ := packet.NewPacket(
		packet.PlayClientboundAnimate,
		mc.VarInt(c.Player.EntityID),
		mc.UnsignedByte(animationID),
	)
	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}

func (c *Connection) HandleChat(pkt *packet.Packet) {
	var message mc.String
	var timestamp, salt mc.Long
	var signature = mc.NewPrefixedOptional(mc.NewArray[mc.Byte](256))
	var messageCount mc.VarInt
	var acknowledged = mc.NewFixedBitSet(20)
	var checksum mc.Byte

	if err := pkt.Decode(&message, &timestamp, &salt, signature, &messageCount, acknowledged, &checksum); err != nil {
		log.Printf("Error decoding chat packet: %v", err)
	}

	// turbo dummy implementation
	type ChatMessage struct {
		Translate string `nbt:"translate"`
		With      []struct {
			Text       string `nbt:"text"`
			ClickEvent *struct {
				Action string `nbt:"action"`
				Value  string `nbt:"value"`
			} `nbt:"clickEvent,omitempty"`
			HoverEvent *struct {
				Action   string `nbt:"action"`
				Contents string `nbt:"contents"`
			} `nbt:"hoverEvent,omitempty"`
		} `nbt:"with"`
	}

	data := ChatMessage{
		Translate: "chat.type.text",
		With: []struct {
			Text       string `nbt:"text"`
			ClickEvent *struct {
				Action string `nbt:"action"`
				Value  string `nbt:"value"`
			} `nbt:"clickEvent,omitempty"`
			HoverEvent *struct {
				Action   string `nbt:"action"`
				Contents string `nbt:"contents"`
			} `nbt:"hoverEvent,omitempty"`
		}{
			{
				Text: string(c.Player.Name),
				ClickEvent: &struct {
					Action string `nbt:"action"`
					Value  string `nbt:"value"`
				}{
					Action: "suggest_command",
					Value:  fmt.Sprintf("/tell %s ", string(c.Player.Name)),
				},
				HoverEvent: &struct {
					Action   string `nbt:"action"`
					Contents string `nbt:"contents"`
				}{
					Action:   "show_text",
					Contents: "Click to message",
				},
			},
			{
				Text: string(message),
			},
		},
	}

	pkt.Retain()
	_ = pkt.ResetWith(packet.PlayClientboundSystemChat)
	encoder := nbt.NewEncoder(pkt.Buffer)
	encoder.NetworkFormat(true)
	if err := encoder.Encode(&data, ""); err != nil {
		log.Printf("Error encoding chat message: %v", err)
		return
	}
	_ = pkt.Encode(mc.Boolean(false))
	c.server.Broadcaster.Broadcast(pkt)
}

func (c *Connection) HandleChatCommand(pkt *packet.Packet) {
	var command mc.String

	if err := pkt.Decode(&command); err != nil {
		log.Printf("Error decoding chat command packet: %v", err)
		return
	}

	// todo: work on some command handling system
}

func (c *Connection) HandleSetCarriedItem(pkt *packet.Packet) {
	var slot mc.Short

	if err := pkt.Decode(&slot); err != nil {
		log.Printf("Error decoding set carried item packet: %v", err)
		return
	}

	c.Player.SelectedItemSlot = int32(slot)
	inventoryId := mc.HotbarToInternal(int(slot))
	item := c.Player.Inventory.Get(inventoryId)
	glass, _ := mc.GetItemByName("glass")
	testSlot := mc.Slot{
		ItemID: int32(glass.ID),
		Count:  1,
	}
	pkt, _ = packet.NewPacket(
		packet.PlayClientboundSetEquipment,
		mc.VarInt(c.Player.EntityID),
		mc.UnsignedByte(0|1<<7),
		&item,
		mc.UnsignedByte(5),
		&testSlot,
	)
	c.server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
}

func (c *Connection) HandleSetCreativeModeSlot(pkt *packet.Packet) {
	var slot mc.Short
	var slotData mc.Slot

	if err := pkt.Decode(&slot, &slotData); err != nil {
		log.Printf("Error decoding set creative mode slot packet: %v", err)
		return
	}

	_ = c.Player.Inventory.Set(int(slot), slotData)
}

func (c *Connection) HandleUseItemOn(pkt *packet.Packet) {
	var hand, face, sequence mc.VarInt
	var location mc.Position
	var cursorX, cursorY, cursorZ mc.Float
	var insideBlock, worldBorderHit mc.Boolean

	if err := pkt.Decode(&hand, &location, &face, &cursorX, &cursorY, &cursorZ, &insideBlock, &worldBorderHit, &sequence); err != nil {
		log.Printf("Error decoding use item on packet: %v", err)
		return
	}

	switch face {
	case 0: // Bottom
		location.Y--
	case 1: // Top
		location.Y++
	case 2: // North
		location.Z--
	case 3: // South
		location.Z++
	case 4: // West
		location.X--
	case 5: // East
		location.X++
	}

	var slotId = mc.HotbarToInternal(int(c.Player.SelectedItemSlot))
	var slotData = c.Player.Inventory.Get(slotId)

	if slotData.Count > 0 {
		item, ok := mc.GetItem(int(slotData.ItemID))

		if ok && item.BlockID != -1 {
			block, _ := mc.GetBlock(item.BlockID)
			pkt, _ = packet.NewPacket(
				packet.PlayClientboundBlockUpdate,
				location,
				mc.VarInt(block.DefaultStateID),
			)
			c.server.Broadcaster.Broadcast(pkt)

			// todo: fix to handle sound groups
			if soundId, ok := block.Sounds["place"]; ok {
				soundPkt, _ := packet.NewPacket(
					packet.PlayClientboundSound,
					mc.VarInt(soundId+1),
					mc.VarInt(4),
					mc.Int(location.X*8),
					mc.Int(location.Y*8),
					mc.Int(location.Z*8),
					mc.Float(1),
					mc.Float(1),
					mc.Long(0),
				)
				c.server.Broadcaster.Broadcast(soundPkt, systems.NotSender(c))
			}
		}
	}

	pkt, _ = packet.NewPacket(packet.PlayClientboundBlockChangedAck, sequence)
	c.Send(pkt)
}
