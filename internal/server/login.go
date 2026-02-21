package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/google/uuid"
)

type GameProfile struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Properties []struct {
		Name      string `json:"name"`
		Value     string `json:"value"`
		Signature string `json:"signature,omitempty"`
	} `json:"properties"`
}

func (c *Connection) HandleLoginStartPacket(pkt *packet.Packet) {
	var (
		Name       mc.String
		PlayerUUID mc.UUID
	)

	if err := pkt.Decode(&Name, &PlayerUUID); err != nil {
		logger.Error("Error decoding loginStart packet: %v", err)
		return
	}

	ip := c.Conn.RemoteAddr().String()
	if banned, entry := c.server.AccessControl.IsIPBanned(ip); banned {
		c.Disconnect(tc.Text(fmt.Sprintf("You are IP banned: %s", entry.Reason)))
		return
	}

	if banned, entry := c.server.AccessControl.IsBanned(uuid.UUID(PlayerUUID)); banned {
		c.Disconnect(tc.Text(fmt.Sprintf("You are banned: %s", entry.Reason)))
		return
	}

	if c.server.Properties.WhiteList && !c.server.AccessControl.IsWhitelisted(uuid.UUID(PlayerUUID)) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
		return
	}

	c.server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player != nil && conn.Player.UUID == uuid.UUID(PlayerUUID) {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectDuplicateLogin))
			return false
		}
		return true
	})

	url := "https://sessionserver.mojang.com/session/minecraft/profile/" + uuid.UUID(PlayerUUID).String()
	if c.server.Properties.OnlineMode == true {
		url += "?unsigned=false"
	}
	res, err := http.Get(url)
	if err != nil {
		logger.Error("Error fetching player data from Mojang API: %v", err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		logger.Error("Mojang API returned non-200 status code: %d", res.StatusCode)
	}

	var profile GameProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		logger.Error("Error parsing player data from Mojang API: %v", err)
	}

	var profileProperties = make([]mc.ProfileProperty, 0, len(profile.Properties))

	_ = pkt.ResetWith(packet.LoginClientboundLoginFinished, &PlayerUUID, &Name)
	_ = pkt.Encode(mc.VarInt(len(profile.Properties)))
	for _, prop := range profile.Properties {
		newProperty := mc.ProfileProperty{
			Name:      prop.Name,
			Value:     prop.Value,
			Signature: prop.Signature,
		}
		profileProperties = append(profileProperties, newProperty)
		_ = pkt.Encode(newProperty)
	}
	_ = pkt.Send(c.Conn)

	newPlayer := entities.NewPlayer(uuid.UUID(PlayerUUID), string(Name), profileProperties, c.server.Properties)
	newPlayer.ProfileProperties = profileProperties
	c.Player = newPlayer
}

func (c *Connection) HandleLoginAckPacket(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundSelectKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn)
}
