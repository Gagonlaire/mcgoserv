package server

import (
	"bytes"
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
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/google/uuid"
)

func (c *Connection) HandleLoginStart(data *decoders.LoginStart) {
	c.ContextData["loginName"] = string(data.Name)
	logger.Debug("Login request from %s (%s)", string(data.Name), c.Conn.RemoteAddr())
	if !c.Server.Config.Security.OnlineMode {
		offlineUUID := internal.GetOfflineUUID(c.ContextData["loginName"].(string))
		c.ContextData["loginUUID"] = offlineUUID
		logger.Debug("Offline mode: assigned UUID %s to %s", offlineUUID, string(data.Name))
		c.FinishLogin([]api.MojangSessionProperty{})
		return
	}

	verifyToken := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, verifyToken); err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		logger.Error("Error generating verify token: %v", err)
		return
	}
	c.ContextData["verifyToken"] = verifyToken

	pArrayPublicKey := mc.NewPrefixedByteArray(c.Server.Keys.EncodedPublicKey)
	pArrayVerifyToken := mc.NewPrefixedByteArray(verifyToken)
	pkt := c.NewPacket(
		packet.LoginClientboundHello,
		mc.String(c.Server.ID),
		pArrayPublicKey,
		pArrayVerifyToken,
		mc.Boolean(true),
	)
	c.SendSync(pkt)
}

func (c *Connection) HandleEncryptionResponse(data *decoders.EncryptionResponse) {
	// As of 1.21, the vanilla server never uses encryption in offline mode.
	if !c.Server.Config.Security.OnlineMode {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPacket))
		return
	}

	decryptedSecret, err := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Keys.PrivateKey, data.EncryptedSecret.Data)
	if err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	decryptedVerifyToken, err := rsa.DecryptPKCS1v15(rand.Reader, c.Server.Keys.PrivateKey, data.EncryptedVerifyToken.Data)
	if err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectGeneric))
		return
	}
	if !bytes.Equal(decryptedVerifyToken, c.ContextData["verifyToken"].([]byte)) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectUnverifiedUsername))
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
	logger.Debug("Encryption enabled for %s", c.Conn.RemoteAddr())

	authHash := internal.AuthDigest(c.Server.ID + string(decryptedSecret) + string(c.Server.Keys.EncodedPublicKey))
	url := fmt.Sprintf("https://sessionserver.mojang.com/session/minecraft/hasJoined?username=%s&serverId=%s", c.ContextData["loginName"].(string), authHash)
	if c.Server.Config.Security.PreventProxyConnection {
		url += "&ip=" + c.Conn.RemoteAddr().String()
	}

	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Failed to contact session server: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectAuthserversDown))
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		logger.Error("Session server returned status %d: %s", resp.StatusCode, string(body))
		if resp.StatusCode >= 500 {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectAuthserversDown))
		} else {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectUnverifiedUsername))
		}
		return
	}

	var session api.MojangSession
	if err := json.Unmarshal(body, &session); err != nil {
		logger.Error("Failed to parse session server response: %v", err)
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectUnverifiedUsername))
		return
	}

	realUUID, _ := uuid.Parse(session.ID)
	c.ContextData["loginUUID"] = realUUID
	c.ContextData["loginName"] = session.Name
	c.FinishLogin(session.Properties)
}

func (c *Connection) FinishLogin(properties []api.MojangSessionProperty) {
	if !c.CanAccessServer() {
		return
	}

	permissionLevel := 0
	if ok, opEntry := c.Server.PlayerRegistry.IsOp(c.ContextData["loginUUID"].(uuid.UUID)); ok {
		permissionLevel = opEntry.Level
	}
	pArraySession := mc.MapToPrefixedArray[mc.ProfileProperty, *mc.ProfileProperty](properties, func(p api.MojangSessionProperty) mc.ProfileProperty {
		return mc.ProfileProperty{
			Name:      p.Name,
			Value:     p.Value,
			Signature: p.Signature,
		}
	})
	loginUUID := c.ContextData["loginUUID"].(uuid.UUID)
	loginName := c.ContextData["loginName"].(string)
	logger.Info("UUID of player %s is %s", logger.Identity(loginName), logger.Identity(loginUUID))

	c.Player = entities.NewPlayer(
		loginUUID,
		loginName,
		permissionLevel,
		pArraySession.Data,
		c.Server.Config,
	)

	if c.Server.Config.Network.Compression.Enabled {
		logger.Debug("Enabling compression (threshold=%d) for %s", c.Server.Config.Network.Compression.Threshold, loginName)
		pkt := c.NewPacket(packet.LoginClientboundLoginCompression, mc.VarInt(c.Server.Config.Network.Compression.Threshold))
		c.SendSync(pkt)
		c.CompressionThreshold = c.Server.Config.Network.Compression.Threshold
	}

	pkt := c.NewPacket(
		packet.LoginClientboundLoginFinished,
		mc.UUID(loginUUID),
		mc.String(loginName),
		pArraySession,
	)
	c.SendSync(pkt)
	delete(c.ContextData, "loginName")
	delete(c.ContextData, "loginUUID")
}

func (c *Connection) CanAccessServer() bool {
	UUID := c.ContextData["loginUUID"].(uuid.UUID)

	if banned, entry := c.Server.PlayerRegistry.IsIPBanned(c.Conn.RemoteAddr().String()); banned {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedIpReason, tc.Text(entry.Reason)))
		return false
	}

	if banned, entry := c.Server.PlayerRegistry.IsBanned(UUID); banned {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBannedReason, tc.Text(entry.Reason)))
		return false
	}

	if c.Server.Config.Security.Whitelist.Enabled && !c.Server.PlayerRegistry.IsWhitelisted(UUID) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectNotWhitelisted))
		return false
	}

	if !c.Server.Config.Security.OnlineMode {
		c.Server.PlayerRegistry.ReconcileWhitelistName(UUID, c.ContextData["loginName"].(string))
	}

	player := c.Server.World.PlayersByUUID[UUID]
	isRejoining := player != nil

	if c.Server.World.OnlinePlayersCount() >= c.Server.Config.Server.MaxPlayers && !isRejoining {
		if op, entry := c.Server.PlayerRegistry.IsOp(UUID); !op || !entry.BypassesPlayerLimit {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectServerFull))
			return false
		}
	}

	if player != nil {
		logger.Debug("Disconnecting existing session for %s (duplicate login)", player.Name)
		c.Server.Connections.Range(func(k, v interface{}) bool {
			conn := k.(*Connection)
			if conn.Player == player {
				conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectDuplicateLogin))
				return false
			}
			return true
		})
	}

	return true
}

func (c *Connection) HandleLoginAcknowledged(_ *packet.InboundPacket) {
	c.State = mc.StateConfiguration
	c.LastKeepAlive = c.Server.World.Time
	logger.Debug("%s entering configuration state", c.Player.Name)

	pkt := c.NewPacket(packet.ConfigurationClientboundSelectKnownPacks, mc.ServerDataPacks)
	c.SendSync(pkt)
}
