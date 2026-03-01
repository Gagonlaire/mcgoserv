package server

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Gagonlaire/mcgoserv/internal"
	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/google/uuid"
)

func (c *Connection) HandleLoginStart(pkt *packet.Packet) {
	var (
		Name       mc.String
		PlayerUUID mc.UUID
	)

	if err := pkt.Decode(&Name, &PlayerUUID); err != nil {
		logger.Error("Error decoding loginStart packet: %v", err)
		return
	}

	c.ContextData["loginName"] = string(Name)

	verifyToken := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, verifyToken); err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		logger.Error("Error generating verify token: %v", err)
		return
	}
	c.ContextData["verifyToken"] = verifyToken

	pArrayPublicKey := mc.NewPrefixedArrayFromSlice(c.Server.Keys.EncodedPublicKey, func(b byte) mc.Byte { return mc.Byte(b) })
	pArrayVerifyToken := mc.NewPrefixedArrayFromSlice(verifyToken, func(b byte) mc.Byte { return mc.Byte(b) })
	_ = pkt.ResetWith(
		packet.LoginClientboundHello,
		mc.String(c.Server.ID),
		pArrayPublicKey,
		pArrayVerifyToken,
		mc.Boolean(c.Server.Properties.OnlineMode),
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}

func (c *Connection) HandleLoginEncryptionResponse(pkt *packet.Packet) {
	var encryptedSecret, encryptedVerifyToken mc.PrefixedArray[mc.Byte]

	if err := pkt.Decode(&encryptedSecret, &encryptedVerifyToken); err != nil {
		logger.Error("Error decoding encryption response packet: %v", err)
		return
	}

	decryptedSecret, _ := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Keys.PrivateKey, mc.MapToSlice(&encryptedSecret, func(b mc.Byte) byte { return byte(b) }))
	decryptedVerifyToken, _ := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Keys.PrivateKey, mc.MapToSlice(&encryptedVerifyToken, func(b mc.Byte) byte { return byte(b) }))
	if !internal.EqualBytes(decryptedVerifyToken, c.ContextData["verifyToken"].([]byte)) {
		// todo: replace with correct message
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	delete(c.ContextData, "verifyToken")

	encryptedConn, err := NewEncryptedConn(c.Conn, decryptedSecret)
	if err != nil {
		logger.Error("Failed to enable encryption: %v", err)
		c.close() // Client is already encrypted, we cannot send packets
		return
	}
	c.Conn = encryptedConn

	var session api.MojangSession

	if c.Server.Properties.OnlineMode {
		authHash := internal.AuthDigest(c.Server.ID + string(decryptedSecret) + string(c.Server.Keys.EncodedPublicKey))
		url := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/hasJoined?username=%s&serverId=%s", c.ContextData["loginName"].(string), authHash)
		if c.Server.Properties.PreventProxyConnections {
			url += "&ip=" + c.Conn.RemoteAddr().String()
		}

		resp, err := http.Get(url)
		if err != nil {
			logger.Error("Failed to contact session server: %v", err)
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
			return
		}
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			// todo: match the error message to the status code
			logger.Error("Session server returned status %d: %s", resp.StatusCode, string(body))
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
			return
		}

		if err := json.Unmarshal(body, &session); err != nil {
			logger.Error("Failed to parse session server response: %v", err)
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
			return
		}

		realUUID, _ := uuid.Parse(session.ID)
		c.ContextData["loginUUID"] = realUUID
		c.ContextData["loginName"] = session.Name
	} else {
		offlineUUID := internal.GetOfflineUUID(c.ContextData["loginName"].(string))
		c.ContextData["loginUUID"] = offlineUUID
		session = api.MojangSession{
			ID:         offlineUUID.String(),
			Name:       c.ContextData["loginName"].(string),
			Properties: []api.MojangSessionProperty{},
		}
	}

	if c.CanAccessServer() {
		pArraySession := mc.NewPrefixedArrayFromSlice(session.Properties, func(p api.MojangSessionProperty) mc.ProfileProperty {
			return mc.ProfileProperty{
				Name:      p.Name,
				Value:     p.Value,
				Signature: p.Signature,
			}
		})

		newPlayer := entities.NewPlayer(
			c.ContextData["loginUUID"].(uuid.UUID),
			c.ContextData["loginName"].(string),
			*pArraySession.Slice,
			c.Server.Properties,
		)
		c.Player = newPlayer
		c.FinishLogin(pArraySession)
	}
}

func (c *Connection) CanAccessServer() bool {
	UUID := c.ContextData["loginUUID"].(uuid.UUID)

	if banned, entry := c.Server.AccessControl.IsIPBanned(c.Conn.RemoteAddr().String()); banned {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedIpReason, tc.Text(entry.Reason)))
		return false
	}

	if banned, entry := c.Server.AccessControl.IsBanned(UUID); banned {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedReason, tc.Text(entry.Reason)))
		return false
	}

	if c.Server.Properties.WhiteList && !c.Server.AccessControl.IsWhitelisted(UUID) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
		return false
	}

	isRejoining := false
	c.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		if conn.Player != nil && conn.Player.UUID == c.ContextData["loginUUID"] {
			isRejoining = true
			return false
		}
		return true
	})

	if len(c.Server.World.Players) >= c.Server.Properties.MaxPlayers && !isRejoining {
		if op, entry := c.Server.AccessControl.IsOp(UUID); !op || !entry.BypassesPlayerLimit {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectServerFull))
			return false
		}
	}

	c.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		if conn.Player != nil && conn.Player.UUID == c.ContextData["loginUUID"] {
			conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectDuplicateLogin))
		}
		return true
	})

	return true
}

func (c *Connection) FinishLogin(properties *mc.PrefixedArray[mc.ProfileProperty]) {
	if c.Server.Properties.NetworkCompressionThreshold >= 0 {
		pkt, _ := packet.NewPacket(packet.LoginClientboundLoginCompression, mc.VarInt(c.Server.Properties.NetworkCompressionThreshold))
		_ = pkt.Send(c.Conn, c.CompressionThreshold)
		c.CompressionThreshold = c.Server.Properties.NetworkCompressionThreshold
	}

	UUID := mc.UUID(c.ContextData["loginUUID"].(uuid.UUID))
	pkt, _ := packet.NewPacket(
		packet.LoginClientboundLoginFinished,
		&UUID,
		mc.String(c.ContextData["loginName"].(string)),
		properties,
	)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	delete(c.ContextData, "loginName")
	delete(c.ContextData, "loginUUID")
}

func (c *Connection) HandleLoginAck(pkt *packet.Packet) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.Server.World.Time

	_ = pkt.ResetWith(packet.ConfigurationClientboundSelectKnownPacks, &mc.ServerDataPacks)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}
