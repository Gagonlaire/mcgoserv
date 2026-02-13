package packet

import (
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

func BuildPlayerInfoUpdatePacket(actions mc.PlayerAction, players []*world.Player) (*Packet, error) {
	// todo: i'm not sure this should be here
	packet, _ := NewPacket(PlayClientboundPlayerInfoUpdate)

	_ = packet.Encode(&actions)
	playerCount := mc.VarInt(len(players))
	_ = packet.Encode(&playerCount)
	for _, player := range players {
		uuid := mc.UUID(player.UUID)
		_ = packet.Encode(&uuid)

		for bit := 0; bit < 8; bit++ {
			currentAction := mc.PlayerAction(1 << bit)

			if actions&currentAction != 0 {
				switch currentAction {
				case mc.ActionAddPlayer:
					name := mc.String(player.Name)
					propCount := mc.VarInt(0)

					_ = packet.Encode(&name, &propCount)
				case mc.ActionUpdateListed:
					// todo: replace with real value
					listed := mc.Boolean(true)
					_ = packet.Encode(&listed)
				}
			}
		}
	}

	return packet, nil
}
