package server

import (
	"crypto/x509"
	"math/rand"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
)

func (c *Connection) HandleKeepAlive(id *mc.Long) {
	c.LastKeepAliveID = int64(*id)
	c.LastKeepAlive = c.Server.World.Time
}

func (c *Connection) SendSpawnEntity(entity *world.Entity) {
	// todo: check for head/body rotation
	pkt := c.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(entity))
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
	pkt := c.NewPacket(packetId, mc.Long(c.Server.World.Time))
	c.Send(pkt)
}

func (c *Connection) HandlePlayerInput(flags *mc.UnsignedByte) {
	c.Player.Input = byte(*flags)

	if (*flags)&mc.InputSneak != 0 {
		pkt2 := c.NewPacket(
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
		c.Server.BroadcastViewers(c, pkt2)
	} else {
		pkt2 := c.NewPacket(
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
		c.Server.BroadcastViewers(c, pkt2)
	}
}

func (c *Connection) HandlePlayerLoaded(_ *packet.InboundPacket) {
	c.Player.Loaded = true
}

func (c *Connection) HandlePlayerCommand(data *decoders.PlayerCommand) {
	// todo: jumping seems to stop sprinting animation particles
	switch mc.PlayerCommand(data.ActionID) {
	case mc.ActionStartSprinting:
		pkt2 := c.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0x08),
			mc.UnsignedByte(0xff),
		)
		c.Server.BroadcastViewers(c, pkt2)
	case mc.ActionStopSprinting:
		pkt2 := c.NewPacket(
			packet.PlayClientboundSetEntityData,
			mc.VarInt(c.Player.EntityID),
			mc.UnsignedByte(0),
			mc.VarInt(0),
			mc.Byte(0),
			mc.UnsignedByte(0xff),
		)
		c.Server.BroadcastViewers(c, pkt2)
	}
}

func (c *Connection) HandleSwingArm(hand *mc.VarInt) {
	var animationID int

	if *hand == 0 {
		animationID = 0
	} else {
		animationID = 3
	}

	c.AnimateEntity(animationID)
}

func (c *Connection) HandlePlayerAction(data *decoders.PlayerAction) {
	switch data.Status {
	case mc.StatusStartDigging:
		if c.Player.GameMode == 1 {
			dim := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
			blockState, _ := dim.GetBlock(int(data.Location.X), int(data.Location.Y), int(data.Location.Z))

			_ = dim.SetBlock(int(data.Location.X), int(data.Location.Y), int(data.Location.Z), 0)
			pkt := c.NewPacket(
				packet.PlayClientboundBlockUpdate,
				data.Location,
				mc.VarInt(0),
			)
			eventPkt := c.NewPacket(
				packet.PlayClientboundLevelEvent,
				mc.Int(2001),
				data.Location,
				mc.Int(blockState),
				mc.Boolean(false),
			)
			c.Server.BroadcastOthers(c, eventPkt)
			c.Server.BroadcastAll(pkt)
		}
	case mc.StatusFinishDigging:
		pkt := c.NewPacket(
			packet.PlayClientboundBlockUpdate,
			data.Location,
			mc.VarInt(0),
		)
		c.Server.BroadcastAll(pkt)
	}

	pkt := c.NewPacket(packet.PlayClientboundBlockChangedAck, data.Sequence)
	c.Send(pkt)
}

func (c *Connection) AnimateEntity(animationID int) {
	pkt := c.NewPacket(
		packet.PlayClientboundAnimate,
		mc.VarInt(c.Player.EntityID),
		mc.UnsignedByte(animationID),
	)
	c.Server.BroadcastViewers(c, pkt)
}

func (c *Connection) HandleSetHeldItem(slot *mc.Short) {
	c.Player.SelectedItemSlot = int32(*slot)
	inventoryId := mc.HotbarToInternal(int(*slot))
	item := c.Player.Inventory.Get(inventoryId)
	if item.Count > 0 {
		pkt := c.NewPacket(
			packet.PlayClientboundSetEquipment,
			mc.VarInt(c.Player.EntityID),
			// todo: check item slot to know if main or off hand
			mc.UnsignedByte(0),
			&item,
		)
		c.Server.BroadcastViewers(c, pkt)
	}
}

func (c *Connection) HandleSetCreativeModeSlot(data *decoders.SetCreativeModeSlot) {
	_ = c.Player.Inventory.Set(int(data.Slot), data.ClickedItem)
}

func (c *Connection) HandleUseItemOn(data *decoders.UseItemOn) {
	switch data.Face {
	case 0: // Bottom
		data.Location.Y--
	case 1: // Top
		data.Location.Y++
	case 2: // North
		data.Location.Z--
	case 3: // South
		data.Location.Z++
	case 4: // West
		data.Location.X--
	case 5: // East
		data.Location.X++
	}

	var slotId = mc.HotbarToInternal(int(c.Player.SelectedItemSlot))
	var slotData = c.Player.Inventory.Get(slotId)

	if slotData.Count > 0 {
		item, ok := mcdata.GetItem(int(slotData.ItemID))

		if ok && item.BlockID != -1 {
			block, _ := mcdata.GetBlock(item.BlockID)
			dim := world.GetEntityDimension(&c.Player.LivingEntity.BaseEntity)
			_ = dim.SetBlock(int(data.Location.X), int(data.Location.Y), int(data.Location.Z), int32(block.DefaultStateID))

			pkt := c.NewPacket(
				packet.PlayClientboundBlockUpdate,
				data.Location,
				mc.VarInt(block.DefaultStateID),
			)
			c.Server.BroadcastAll(pkt)

			// todo: fix to handle sound groups
			// todo: check if faster rand exist
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			// todo: sounds weird, tweak values
			pitch := 0.5 + r.Float64()*(2-0.5)
			if soundId, ok := block.Sounds["place"]; ok {
				soundPkt := c.NewPacket(
					packet.PlayClientboundSound,
					mc.VarInt(soundId+1),
					mc.VarInt(4),
					mc.Int(data.Location.X*8),
					mc.Int(data.Location.Y*8),
					mc.Int(data.Location.Z*8),
					mc.Float(1),
					mc.Float(pitch),
					mc.Long(0),
				)
				c.Server.BroadcastOthers(c, soundPkt)
			}
		}
	}

	pkt := c.NewPacket(packet.PlayClientboundBlockChangedAck, data.Sequence)
	c.Send(pkt)
}

func buildPlayerInfoUpdatePacket(actions mc.PlayerAction, players []*entities.Player) (*packet.OutboundPacket, error) {
	pkt, err := packet.NewPacket(packet.PlayClientboundPlayerInfoUpdate)
	if err != nil {
		return nil, err
	}
	playerCount := mc.VarInt(len(players))

	_ = pkt.Encode(actions, playerCount)
	for _, player := range players {
		_ = pkt.Encode(mc.UUID(player.UUID))

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
						pubKeyBytes, err := x509.MarshalPKIXPublicKey(player.ChatSession.PublicKey)
						if err != nil {
							pubKeyBytes = []byte{}
						}
						pArrayPublicKey := mc.NewPrefixedByteArray(pubKeyBytes)
						pArraySignature := mc.NewPrefixedByteArray(player.ChatSession.KeySignature)
						_ = pkt.Encode(
							mc.UUID(player.ChatSession.ID),
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
