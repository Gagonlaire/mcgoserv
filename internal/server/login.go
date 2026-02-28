package server

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net/http"

	"github.com/Gagonlaire/mcgoserv/internal"
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

func (c *Connection) HandleLoginStart(pkt *packet.Packet) {
	var (
		Name       mc.String
		PlayerUUID mc.UUID
	)

	if err := pkt.Decode(&Name, &PlayerUUID); err != nil {
		logger.Error("Error decoding loginStart packet: %v", err)
		return
	}

	c.TempName = string(Name)
	c.TempUUID = uuid.UUID(PlayerUUID)
	ip := c.Conn.RemoteAddr().String()
	if banned, entry := c.Server.AccessControl.IsIPBanned(ip); banned {
		c.Disconnect(tc.Text(fmt.Sprintf("You are IP banned: %s", entry.Reason)))
		return
	}

	if banned, entry := c.Server.AccessControl.IsBanned(uuid.UUID(PlayerUUID)); banned {
		c.Disconnect(tc.Text(fmt.Sprintf("You are banned: %s", entry.Reason)))
		return
	}

	if c.Server.Properties.WhiteList && !c.Server.AccessControl.IsWhitelisted(uuid.UUID(PlayerUUID)) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
		return
	}

	c.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)

		if conn.Player != nil && conn.Player.UUID == uuid.UUID(PlayerUUID) {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectDuplicateLogin))
			return false
		}
		return true
	})

	if c.Server.Properties.OnlineMode {
		verifyToken := make([]byte, 16)
		if _, err := io.ReadFull(rand.Reader, verifyToken); err != nil {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
			logger.Error("Error generating verify token: %v", err)
			return
		}
		c.VerifyToken = verifyToken

		pArrayPublicKey := mc.NewPrefixedArrayFromSlice(c.Server.EncodedPublicKey, func(b byte) mc.Byte { return mc.Byte(b) })
		pArrayVerifyToken := mc.NewPrefixedArrayFromSlice(verifyToken, func(b byte) mc.Byte { return mc.Byte(b) })
		_ = pkt.ResetWith(
			packet.LoginClientboundHello,
			mc.String(c.Server.ID),
			pArrayPublicKey,
			pArrayVerifyToken,
			mc.Boolean(true),
		)
		_ = pkt.Send(c.Conn, c.CompressionThreshold)
	} else {
		c.FinishLogin()
	}
}

func (c *Connection) HandleLoginEncryptionResponse(pkt *packet.Packet) {
	var encryptedSecret, encryptedVerifyToken mc.PrefixedArray[mc.Byte]

	if err := pkt.Decode(&encryptedSecret, &encryptedVerifyToken); err != nil {
		logger.Error("Error decoding encryption response packet: %v", err)
		return
	}

	decryptedSecret, _ := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Key, mc.MapToSlice(&encryptedSecret, func(b mc.Byte) byte { return byte(b) }))
	decryptedVerifyToken, _ := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Key, mc.MapToSlice(&encryptedVerifyToken, func(b mc.Byte) byte { return byte(b) }))

	if !internal.EqualBytes(decryptedVerifyToken, c.VerifyToken) {
		// todo: replace with correct message
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}

	encryptedConn, err := NewEncryptedConn(c.Conn, decryptedSecret)
	if err != nil {
		logger.Error("Failed to enable encryption: %v", err)
		c.close() // Client is already encrypted, we cannot send packets
		return
	}
	c.Conn = encryptedConn

	authHash := internal.AuthDigest(c.Server.ID + string(decryptedSecret) + string(c.Server.EncodedPublicKey))
	url := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/hasJoined?username=%s&serverId=%s", c.TempName, authHash)
	if c.Server.Properties.PreventProxyConnections {
		url += "&ip=" + c.Conn.RemoteAddr().String()
	}

	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Failed to contact session server: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// todo: match the error message to the status code
		body, _ := io.ReadAll(resp.Body)
		logger.Error("Session server returned status %d: %s", resp.StatusCode, string(body))
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}

	c.FinishLogin()
}

func (c *Connection) HandleLoginAck(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.Server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundSelectKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}

func (c *Connection) FinishLogin() {
	if c.Server.Properties.NetworkCompressionThreshold >= 0 {
		pkt, _ := packet.NewPacket(packet.LoginClientboundLoginCompression, mc.VarInt(c.Server.Properties.NetworkCompressionThreshold))
		_ = pkt.Send(c.Conn, c.CompressionThreshold)
		c.CompressionThreshold = c.Server.Properties.NetworkCompressionThreshold
	}

	/*url := "https://sessionserver.mojang.com/session/minecraft/profile/" + uuid.UUID(PlayerUUID).String()
	if c.Server.Properties.OnlineMode == true {
		url += "?unsigned=false"
	}
	res, err := http.Get(url)
	if err != nil {
		logger.Error("Error fetching player data from Mojang API: %v", err)
	}
	body, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
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
	_ = pkt.Send(c.Conn, c.CompressionThreshold)*/

	UUID := mc.UUID(c.TempUUID)
	pkt, _ := packet.NewPacket(packet.LoginClientboundLoginFinished, &UUID, mc.String(c.TempName), mc.VarInt(0))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)

	emptyProfileProperties := make([]mc.ProfileProperty, 0)
	newPlayer := entities.NewPlayer(uuid.UUID(UUID), c.TempName, emptyProfileProperties, c.Server.Properties)
	newPlayer.ProfileProperties = emptyProfileProperties
	c.Player = newPlayer
}
