package server

import (
	"crypto/x509"
	"math/rand"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/block"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
)

func (c *Connection) HandleKeepAlive(id *mc.Long) {
	if !c.acceptKeepAlive(int64(*id)) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPacket))
	}
}

func (c *Connection) acceptKeepAlive(id int64) bool {
	if !c.KeepAlivePending || id != c.LastKeepAliveID {
		return false
	}
	c.KeepAlivePending = false
	return true
}

func (c *Connection) keepAliveState(now time.Time) (send, timedOut bool) {
	if c.State != mc.StatePlay && c.State != mc.StateConfiguration {
		return false, false
	}
	if c.KeepAlivePending {
		return false, now.Sub(c.LastKeepAliveSent) > c.Server.KeepAliveTimeout
	}
	return now.Sub(c.LastKeepAliveSent) >= c.Server.KeepAliveInterval, false
}

func (c *Connection) SendSpawnEntity(entity entity.Entity) {
	// todo: check for head/body rotation
	pkt := c.NewPacket(packet.PlayClientboundAddEntity, encoders.NewAddEntity(entity))
	c.Send(pkt)
}

// SendChunkEntities spawns every entity already present in the chunk for this connection, skipping self.
func (c *Connection) SendChunkEntities(chunk *mc.Chunk) {
	selfID := c.Player.EntityID
	for entityID := range chunk.Entities {
		if entityID == selfID {
			continue
		}
		if entity := c.Server.World.EntitiesByID[entityID]; entity != nil {
			c.SendSpawnEntity(entity)
		}
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

	now := time.Now()
	c.LastKeepAliveID = now.UnixMilli()
	c.LastKeepAliveSent = now
	c.KeepAlivePending = true
	pkt := c.NewPacket(packetId, mc.Long(c.LastKeepAliveID))
	c.Send(pkt)
}

func (c *Connection) HandlePlayerInput(flags *mc.UnsignedByte) {
	c.Player.Input = byte(*flags)

	// NOTE mainly used for vehicle control
	if mc.PlayerInput(*flags)&mc.InputSneak != 0 {
		c.Player.SetFlag(entity.FlagCrouching, true)
		c.Player.SetPose(entity.PoseSneaking)
	} else {
		c.Player.SetFlag(entity.FlagCrouching, false)
		c.Player.SetPose(entity.PoseStanding)
	}
	c.Server.World.EnqueueDirty(c.Player)
}

func (c *Connection) HandlePlayerLoaded(_ *packet.InboundPacket) {
	c.Player.Loaded = true
}

func (c *Connection) HandleClientCommand(action *mc.VarInt) {
	switch *action {
	case 0:
		c.Server.Respawn(c.Player)
	}
}

func (c *Connection) HandlePlayerCommand(data *decoders.PlayerCommand) {
	switch mc.PlayerCommand(data.ActionID) {
	case mc.CommandStartSprinting:
		c.Player.SetFlag(entity.FlagSprinting, true)
	case mc.CommandStopSprinting:
		c.Player.SetFlag(entity.FlagSprinting, false)
	}
	c.Server.World.EnqueueDirty(c.Player)
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
	case mc.VarInt(mc.ActionStartDigging):
		if c.Player.GameMode == 1 {
			dim := c.Server.World.GetEntityDimension(c.Player)
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
	case mc.VarInt(mc.ActionFinishDigging):
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
		itemID, ok := item.FromID(int(slotData.ItemID))

		if ok && itemID.IsBlock() {
			blockID, _ := block.FromID(itemID.BlockID())
			dim := c.Server.World.GetEntityDimension(c.Player)
			_ = dim.SetBlock(int(data.Location.X), int(data.Location.Y), int(data.Location.Z), int32(blockID.DefaultStateID()))

			pkt := c.NewPacket(
				packet.PlayClientboundBlockUpdate,
				data.Location,
				mc.VarInt(blockID.DefaultStateID()),
			)
			c.Server.BroadcastAll(pkt)

			// todo: fix to handle sound groups
			// todo: check if faster rand exist
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			// todo: sounds weird, tweak values
			pitch := 0.5 + r.Float64()*(2-0.5)
			if soundId, ok := blockID.Sounds()["place"]; ok {
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

func buildPlayerInfoUpdatePacket(actions mc.PlayerListAction, players []*entity.Player) (*packet.OutboundPacket, error) {
	pkt, err := packet.NewPacket(packet.PlayClientboundPlayerInfoUpdate)
	if err != nil {
		return nil, err
	}
	playerCount := mc.VarInt(len(players))

	_ = pkt.Encode(mc.UnsignedByte(actions), playerCount)
	for _, player := range players {
		_ = pkt.Encode(mc.UUID(player.UUID))

		for bit := 0; bit < 8; bit++ {
			currentAction := mc.PlayerListAction(1 << bit)

			if actions&currentAction != 0 {
				switch currentAction {
				case mc.ListActionAddPlayer:
					_ = pkt.Encode(mc.String(player.Name), mc.VarInt(len(player.ProfileProperties)))
					for _, prop := range player.ProfileProperties {
						_ = pkt.Encode(prop)
					}
				case mc.ListActionInitializeChat:
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
				case mc.ListActionUpdateListed:
					_ = pkt.Encode(player.Information.AllowServerListings)
				}
			}
		}
	}

	return pkt, nil
}
