package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/world"
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
		fmt.Println("Error decoding loginStart packet:", err)
		return
	}

	ip := c.Conn.RemoteAddr().String()
	if banned, entry := c.server.PlayerList.IsIPBanned(ip); banned {
		c.Disconnect(fmt.Sprintf("You are IP banned: %s", entry.Reason))
		return
	}

	if banned, entry := c.server.PlayerList.IsBanned(uuid.UUID(PlayerUUID)); banned {
		c.Disconnect(fmt.Sprintf("You are banned: %s", entry.Reason))
		return
	}

	if c.server.Properties.WhiteList && !c.server.PlayerList.IsWhitelisted(uuid.UUID(PlayerUUID)) {
		c.Disconnect("You are not whitelisted on this server!")
		return
	}

	c.server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player != nil && conn.Player.UUID == uuid.UUID(PlayerUUID) {
			conn.Disconnect("You have logged in from another location.")
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
		fmt.Println("Error fetching player data from Mojang API:", err)
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Printf("Mojang API returned non-200 status code: %d\n", res.StatusCode)
	}

	var profile GameProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		fmt.Println("Error parsing player data from Mojang API:", err)
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
	c.Player = world.NewPlayer(uuid.UUID(PlayerUUID), Name, profileProperties, c.server.World, c.server.Properties)
}

func (c *Connection) HandleLoginAckPacket(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundSelectKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn)
}
