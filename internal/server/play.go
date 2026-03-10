package server

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/google/uuid"
)

func (c *Connection) HandleKeepAlive(pkt *packet.Packet) {
	var keepAliveId mc.Long

	if err := pkt.Decode(&keepAliveId); err != nil {
		return
	}
	c.LastKeepAliveID = int64(keepAliveId)
	c.LastKeepAlive = c.Server.World.Time
}

func (c *Connection) HandleClientTickEnd(_ *packet.Packet) {
	// Used for some specific logic
}

func (c *Connection) SendSpawnEntity(entity *world.Entity) {
	entityUUID := mc.UUID(entity.UUID)
	yaw := mc.Angle(entity.Rot[0] / 360.0 * 256.0)
	pitch := mc.Angle(entity.Rot[1] / 360.0 * 256.0)
	vel := mc.LpVec3{X: entity.Motion[0], Y: entity.Motion[1], Z: entity.Motion[2]}

	// todo: check for head/body rotation
	pkt, _ := packet.NewPacket(
		packet.PlayClientboundAddEntity,
		mc.VarInt(entity.EntityID),
		&entityUUID,
		mc.VarInt(entity.TypeID),
		mc.Double(entity.Pos[0]), mc.Double(entity.Pos[1]), mc.Double(entity.Pos[2]),
		&vel,
		pitch,
		yaw, // body yaw
		yaw, // head yaw
		mc.VarInt(0),
	)
	c.Send(pkt)
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

	c.LastKeepAliveID = c.Server.World.Time
	pkt, _ := packet.NewPacket(packetId, mc.Long(c.Server.World.Time))
	c.Send(pkt)
}

func (c *Connection) HandlePlayerInput(pkt *packet.Packet) {
	var input mc.UnsignedByte

	if err := pkt.Decode(&input); err != nil {
		log.Printf("Error decoding player input packet: %v", err)
	}

	c.Player.Input = byte(input)

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
		c.Server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
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
		c.Server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	}
}

func (c *Connection) HandlePlayerLoaded(_ *packet.Packet) {
	c.Player.Loaded = true
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
		c.Server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	case mc.ActionStopSprinting:
		pkt2, _ := packet.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0),
			mc.UnsignedByte(0xff),
		)
		c.Server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
	}
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
			dim := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
			blockState, _ := dim.GetBlock(int(location.X), int(location.Y), int(location.Z))

			_ = dim.SetBlock(int(location.X), int(location.Y), int(location.Z), 0)
			pkt, _ := packet.NewPacket(
				packet.PlayClientboundBlockUpdate,
				location,
				mc.VarInt(0),
			)
			eventPkt, _ := packet.NewPacket(
				packet.PlayClientboundLevelEvent,
				mc.Int(2001),
				location,
				mc.Int(blockState),
				mc.Boolean(false),
			)
			c.Server.Broadcaster.Broadcast(eventPkt, systems.NotSender(c))
			c.Server.Broadcaster.Broadcast(pkt)
		}
	case StatusFinishDigging:
		pkt, _ := packet.NewPacket(
			packet.PlayClientboundBlockUpdate,
			location,
			mc.VarInt(0),
		)
		c.Server.Broadcaster.Broadcast(pkt)
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
	c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
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
	if item.Count > 0 {
		pkt, _ = packet.NewPacket(
			packet.PlayClientboundSetEquipment,
			mc.VarInt(c.Player.EntityID),
			// todo: check item slot to know if main or off hand
			mc.UnsignedByte(0),
			&item,
		)
		c.Server.Broadcaster.Broadcast(pkt, systems.NotSender(c))
	}
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
		item, ok := mcdata.GetItem(int(slotData.ItemID))

		if ok && item.BlockID != -1 {
			block, _ := mcdata.GetBlock(item.BlockID)
			dim := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
			_ = dim.SetBlock(int(location.X), int(location.Y), int(location.Z), int32(block.DefaultStateID))

			pkt, _ = packet.NewPacket(
				packet.PlayClientboundBlockUpdate,
				location,
				mc.VarInt(block.DefaultStateID),
			)
			c.Server.Broadcaster.Broadcast(pkt)

			// todo: fix to handle sound groups
			// todo: check if faster rand exist
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			pitch := 0.5 + r.Float64()*(2-0.5)
			if soundId, ok := block.Sounds["place"]; ok {
				soundPkt, _ := packet.NewPacket(
					packet.PlayClientboundSound,
					mc.VarInt(soundId+1),
					mc.VarInt(4),
					mc.Int(location.X*8),
					mc.Int(location.Y*8),
					mc.Int(location.Z*8),
					mc.Float(1),
					mc.Float(pitch),
					mc.Long(0),
				)
				c.Server.Broadcaster.Broadcast(soundPkt, systems.NotSender(c))
			}
		}
	}

	pkt, _ = packet.NewPacket(packet.PlayClientboundBlockChangedAck, sequence)
	c.Send(pkt)
}

func VerifyChatSessionKey(mojangKeys []*rsa.PublicKey, playerUUID uuid.UUID, expiresAt int64, publicKeyBytes []byte, keySignature []byte) error {
	payload := make([]byte, 0, 16+8+len(publicKeyBytes))
	payload = append(payload, playerUUID[:]...)
	payload = binary.BigEndian.AppendUint64(payload, uint64(expiresAt))
	payload = append(payload, publicKeyBytes...)
	hash := sha1.Sum(payload)

	for _, key := range mojangKeys {
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA1, hash[:], keySignature); err == nil {
			return nil
		}
	}
	return fmt.Errorf("key signature could not be verified against any Mojang certificate key")
}

func buildPlayerInfoUpdatePacket(actions mc.PlayerAction, players []*entities.Player) (*packet.Packet, error) {
	pkt, _ := packet.NewPacket(packet.PlayClientboundPlayerInfoUpdate)

	_ = pkt.Encode(&actions)
	playerCount := mc.VarInt(len(players))
	_ = pkt.Encode(&playerCount)
	for _, player := range players {
		UUID := mc.UUID(player.UUID)
		_ = pkt.Encode(&UUID)

		for bit := 0; bit < 8; bit++ {
			currentAction := mc.PlayerAction(1 << bit)

			if actions&currentAction != 0 {
				switch currentAction {
				case mc.ActionAddPlayer:
					_ = pkt.Encode(mc.String(player.Name), mc.VarInt(len(player.ProfileProperties)))
					for _, prop := range player.ProfileProperties {
						_ = pkt.Encode(prop)
					}
				case mc.ActionInitializeChat:
					_ = pkt.Encode(mc.Boolean(player.ChatSession.Signed))
					if player.ChatSession.Signed {
						sessionID := mc.UUID(player.ChatSession.ID)

						pubKeyBytes, err := x509.MarshalPKIXPublicKey(player.ChatSession.PublicKey)
						if err != nil {
							pubKeyBytes = []byte{}
						}
						pArrayPublicKey := mc.NewPrefixedArrayFromSlice(pubKeyBytes, func(b byte) mc.Byte {
							return mc.Byte(b)
						})
						pArraySignature := mc.NewPrefixedArrayFromSlice(player.ChatSession.KeySignature, func(b byte) mc.Byte {
							return mc.Byte(b)
						})
						_ = pkt.Encode(
							&sessionID,
							mc.Long(player.ChatSession.ExpiresAt),
							pArrayPublicKey,
							pArraySignature,
						)
					}
				case mc.ActionUpdateListed:
					_ = pkt.Encode(player.Information.AllowServerListings)
				}
			}
		}
	}

	return pkt, nil
}
