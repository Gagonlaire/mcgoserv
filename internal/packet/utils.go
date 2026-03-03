package packet

import (
	"crypto/x509"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
)

func BuildPlayerInfoUpdatePacket(actions mc.PlayerAction, players []*entities.Player) (*Packet, error) {
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
					_ = packet.Encode(mc.String(player.Name), mc.VarInt(len(player.ProfileProperties)))
					for _, prop := range player.ProfileProperties {
						_ = packet.Encode(prop)
					}
				case mc.ActionInitializeChat:
					_ = packet.Encode(mc.Boolean(player.ChatSession.Signed))
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
						_ = packet.Encode(
							&sessionID,
							mc.Long(player.ChatSession.ExpiresAt),
							pArrayPublicKey,
							pArraySignature,
						)
					}
				case mc.ActionUpdateListed:
					_ = packet.Encode(player.Information.AllowServerListings)
				}
			}
		}
	}

	return packet, nil
}
