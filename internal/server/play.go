package server

import (
	"crypto/x509"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/block"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/mc/item"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/server/encoders"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func (c *Connection) HandleKeepAlive(id *proto.Long) {
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
func (c *Connection) SendChunkEntities(chunk *world.Chunk) {
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
	pkt := c.NewPacket(packetId, proto.Long(c.LastKeepAliveID))
	c.Send(pkt)
}

func (c *Connection) HandlePlayerInput(flags *proto.UnsignedByte) {
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

func (c *Connection) HandleClientCommand(action *proto.VarInt) {
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

func (c *Connection) HandleSwingArm(hand *proto.VarInt) {
	var animationID int

	if *hand == 0 {
		animationID = 0
	} else {
		animationID = 3
	}

	c.AnimateEntity(animationID)
}

func (c *Connection) HandlePlayerAction(data *decoders.PlayerAction) {
	shouldBreak := false
	switch data.Status {
	case proto.VarInt(mc.ActionStartDigging):
		shouldBreak = c.Player.GameMode == 1
	case proto.VarInt(mc.ActionFinishDigging):
		shouldBreak = true
	}

	if shouldBreak {
		c.tryBreakBlock(data.Location)
	}

	pkt := c.NewPacket(packet.PlayClientboundBlockChangedAck, data.Sequence)
	c.Send(pkt)
}

func (c *Connection) tryBreakBlock(loc proto.Position) {
	pos := world.BlockPos{X: int(loc.X), Y: int(loc.Y), Z: int(loc.Z)}
	dim := c.Server.World.GetEntityDimension(c.Player)

	state, _ := dim.GetState(pos)
	blockID, ok := block.FromStateID(int(state))
	if !ok {
		return
	}
	behavior, ok := block.Lookup(blockID)
	if !ok {
		return
	}

	ctx := world.BreakContext{
		Pos:     pos,
		State:   state,
		Breaker: c.Player,
		Tool:    heldStack(c.Player),
	}
	result := dim.BreakBlock(behavior, &ctx)
	c.applyBreakResult(pos, result)
}

func (c *Connection) AnimateEntity(animationID int) {
	pkt := c.NewPacket(
		packet.PlayClientboundAnimate,
		proto.VarInt(c.Player.EntityID),
		proto.UnsignedByte(animationID),
	)
	c.Server.BroadcastViewers(c, pkt)
}

func (c *Connection) HandleSetHeldItem(slot *proto.Short) {
	c.Player.Inventory.SelectedHotbar = int32(*slot)
	held := c.Player.Inventory.Held()
	if held.Count > 0 {
		pkt := c.NewPacket(
			packet.PlayClientboundSetEquipment,
			proto.VarInt(c.Player.EntityID),
			// todo: check item slot to know if main or off hand
			proto.UnsignedByte(0),
			&held,
		)
		c.Server.BroadcastViewers(c, pkt)
	}
}

func (c *Connection) HandleSetCreativeModeSlot(data *decoders.SetCreativeModeSlot) {
	_ = c.Player.Inventory.Set(int(data.Slot), data.ClickedItem)
}

func (c *Connection) HandleUseItemOn(data *decoders.UseItemOn) {
	c.tryPlaceBlock(data)

	pkt := c.NewPacket(packet.PlayClientboundBlockChangedAck, data.Sequence)
	c.Send(pkt)
}

func (c *Connection) tryPlaceBlock(data *decoders.UseItemOn) {
	face := world.Direction(data.Face)
	target := faceOffset(data.Location, face)

	held := c.Player.Inventory.Held()
	if held.Count <= 0 {
		return
	}
	itemID, ok := item.FromID(int(held.ItemID))
	if !ok || !itemID.IsBlock() {
		return
	}
	blockID, ok := block.FromID(itemID.BlockID())
	if !ok {
		return
	}
	behavior, ok := block.Lookup(blockID)
	if !ok {
		return
	}

	ctx := world.PlaceContext{
		Pos:      target,
		Face:     face,
		Hit:      [3]float32{float32(data.CursorPosX), float32(data.CursorPosY), float32(data.CursorPosZ)},
		Player:   c.Player,
		Hand:     entity.Hand(data.Hand),
		UsedItem: heldStack(c.Player),
	}
	dim := c.Server.World.GetEntityDimension(c.Player)
	result := dim.PlaceBlock(behavior, &ctx)
	c.applyPlaceResult(target, result)
}

func faceOffset(loc proto.Position, face world.Direction) world.BlockPos {
	dx, dy, dz := face.Vector()
	return world.BlockPos{
		X: int(loc.X) + dx,
		Y: int(loc.Y) + dy,
		Z: int(loc.Z) + dz,
	}
}

func heldStack(p *entity.Player) item.Stack {
	s := p.Inventory.Held()
	return item.Stack{ID: item.ID(s.ItemID), Count: int(s.Count)}
}

func buildPlayerInfoUpdatePacket(actions mc.PlayerListAction, players []*entity.Player) (*packet.OutboundPacket, error) {
	pkt, err := packet.NewPacket(packet.PlayClientboundPlayerInfoUpdate)
	if err != nil {
		return nil, err
	}
	playerCount := proto.VarInt(len(players))

	_ = pkt.Encode(proto.UnsignedByte(actions), playerCount)
	for _, player := range players {
		_ = pkt.Encode(proto.UUID(player.UUID))

		for bit := 0; bit < 8; bit++ {
			currentAction := mc.PlayerListAction(1 << bit)

			if actions&currentAction != 0 {
				switch currentAction {
				case mc.ListActionAddPlayer:
					_ = pkt.Encode(proto.String(player.Name), proto.VarInt(len(player.ProfileProperties)))
					for _, prop := range player.ProfileProperties {
						_ = pkt.Encode(prop)
					}
				case mc.ListActionInitializeChat:
					_ = pkt.Encode(proto.Boolean(player.ChatSession.Signed))
					if player.ChatSession.Signed {
						pubKeyBytes, err := x509.MarshalPKIXPublicKey(player.ChatSession.PublicKey)
						if err != nil {
							pubKeyBytes = []byte{}
						}
						pArrayPublicKey := proto.NewPrefixedByteArray(pubKeyBytes)
						pArraySignature := proto.NewPrefixedByteArray(player.ChatSession.KeySignature)
						_ = pkt.Encode(
							proto.UUID(player.ChatSession.ID),
							proto.Long(player.ChatSession.ExpiresAt),
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
