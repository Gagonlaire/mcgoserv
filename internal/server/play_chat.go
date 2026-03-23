package server

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/server/decoders"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/google/uuid"
)

const (
	maxChatTimeSkewPast   = 5 * time.Minute
	maxChatTimeSkewFuture = 5 * time.Minute
)

var verifyBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, 4096)
	},
}

func (c *Connection) HandleCommandSuggestion(data *decoders.CommandSuggestionsRequest) {
	input := strings.TrimPrefix(data.Text.Value, "/")
	src := c.playerSource()
	ctx := c.Server.Commander.ParseForSuggestion(src, input)

	if ctx == nil || ctx.Node.SuggestFn == nil {
		resp := c.NewPacket(
			packet.PlayClientboundCommandSuggestions,
			data.TransactionID,
			mc.VarInt(0),
			mc.VarInt(0),
			mc.VarInt(0),
		)
		c.Send(resp)
		return
	}

	startIndex := ctx.Start + 1
	entries := ctx.Node.SuggestFn(src, input[ctx.Start:ctx.Start+ctx.Length])
	resp := c.NewPacket(
		packet.PlayClientboundCommandSuggestions,
		data.TransactionID,
		mc.VarInt(startIndex),
		mc.VarInt(ctx.Length),
		mc.VarInt(len(entries)),
	)
	if resp != nil {
		for _, entry := range entries {
			_ = resp.Encode(mc.String(entry.Text), mc.Boolean(entry.Tooltip != nil))
			if entry.Tooltip != nil {
				_ = resp.Encode(entry.Tooltip)
			}
		}
	}
	c.Send(resp)
}

func (c *Connection) HandlePlayerSession(data *decoders.PlayerSession) {
	if !c.Server.Properties.OnlineMode {
		c.Player.ChatSession.Signed = false
		return
	}

	if time.Now().UnixMilli() > int64(data.ExpiresAt) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectExpiredPublicKey))
		return
	}

	publicKeyBytes := data.PublicKey.Data
	signatureBytes := data.KeySignature.Data

	if err := verifyChatSessionKey(
		c.Server.Keys.CertificateKeys,
		c.Player.UUID,
		int64(data.ExpiresAt),
		publicKeyBytes,
		signatureBytes,
	); err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPublicKeySignature))
		return
	}

	parsedKey, err := x509.ParsePKIXPublicKey(publicKeyBytes)
	if err != nil {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPublicKeySignature))
		return
	}

	rsaKey, ok := parsedKey.(*rsa.PublicKey)
	if !ok {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPublicKeySignature))
		return
	}

	c.Player.ChatSession.ID = uuid.UUID(data.SessionId)
	c.Player.ChatSession.ExpiresAt = int64(data.ExpiresAt)
	c.Player.ChatSession.PublicKey = rsaKey
	c.Player.ChatSession.KeySignature = signatureBytes
	c.Player.ChatSession.Index = -1
	c.Player.ChatSession.Signed = true

	player := []*entities.Player{c.Player}
	pkt, _ := buildPlayerInfoUpdatePacket(mc.ListActionInitializeChat, player)
	c.Server.BroadcastAll(pkt)
}

func (c *Connection) HandleChatCommand(command *mc.String) {
	// todo: commands should maybe ran in a separate routine
	src := c.playerSource()
	_, err := c.Server.Commander.ExecuteInput(
		c.ctx,
		src,
		string(*command),
	)

	if err != nil {
		src.SendMessage(commander.AsCommandError(err).ToComponent())
	}
}

func (c *Connection) HandleSignedChatCommand(data *decoders.SignedChatCommand) {
	chatSession := &c.Player.ChatSession

	if c.Server.EnforceSecureChat && len(data.ArgumentSignatures) > 0 && !chatSession.Signed {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPublicKeySignature))
		return
	}

	if c.Server.EnforceSecureChat && chatSession.Signed && int32(data.MessageCount) > chatSession.LastSeenCount {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBadChatIndex))
		return
	}

	lastSeenSignatures := advanceSession(chatSession, &data.Acknowledged)
	if data.Checksum != computeLastSeenChecksum(lastSeenSignatures) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
		return
	}

	argSigMap := make(map[string][]byte, len(data.ArgumentSignatures))
	for _, argSig := range data.ArgumentSignatures {
		sigBytes := make([]byte, 256)
		copy(sigBytes, argSig.Signature.Data)
		argSigMap[string(argSig.ArgumentName)] = sigBytes
	}

	src := c.playerSource()
	signed := &commander.SignedData{
		ArgSignatures:      argSigMap,
		LastSeenSignatures: lastSeenSignatures,
		Timestamp:          int64(data.Timestamp),
		Salt:               int64(data.Salt),
	}
	parsed := c.Server.Commander.ParseSigned(src, string(data.Command), signed)
	if c.Server.EnforceSecureChat && chatSession.Signed {
		if err := validateSessionTiming(chatSession, int64(data.Timestamp)); err != "" {
			c.Disconnect(tc.Translatable(err))
			return
		}

		for _, node := range parsed.Nodes {
			sigBytes, hasSig := argSigMap[node.Node.Name]
			if !hasSig {
				c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
				return
			}
			argValue := string(data.Command)[node.Range.Start:node.Range.End]
			if !verifyMessageSignature(
				chatSession,
				c.Player.UUID,
				argValue,
				int64(data.Timestamp),
				int64(data.Salt),
				chatSession.Index,
				lastSeenSignatures,
				sigBytes,
			) {
				c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
				return
			}
		}
	}

	// todo: commands should maybe run in a separate goroutine
	_, err := c.Server.Commander.Execute(c.ctx, parsed)
	if err != nil {
		src.SendMessage(commander.AsCommandError(err).ToComponent())
	}
}

func (c *Connection) HandleChatMessage(data *decoders.ChatMessage) {
	chatSession := &c.Player.ChatSession
	isChatMessageSigned := chatSession.Signed && bool(data.Signature.Has)
	if c.Server.EnforceSecureChat {
		if !isChatMessageSigned {
			unsignedErrorPkt := c.NewPacket(
				packet.PlayClientboundSystemChat,
				tc.Text("The server refused to deliver an unsigned message. You can enable chat signing by changing your signing mode or through a prompt screen").SetColor(tc.ColorRed),
				mc.Boolean(false),
			)
			c.Send(unsignedErrorPkt)
			return
		} else if int32(data.MessageCount) > chatSession.LastSeenCount {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBadChatIndex))
			return
		}
	}

	lastSeenSignatures := advanceSession(chatSession, &data.Acknowledged)
	if data.Checksum != computeLastSeenChecksum(lastSeenSignatures) {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
		return
	}

	var signatureBytes []byte
	if isChatMessageSigned {
		signatureBytes = make([]byte, 256)
		copy(signatureBytes, data.Signature.Value.Data)

		if err := validateSessionTiming(chatSession, int64(data.Timestamp)); err != "" {
			c.Disconnect(tc.Translatable(err))
			return
		}
		if !verifyMessageSignature(chatSession, c.Player.UUID, string(data.Message), int64(data.Timestamp), int64(data.Salt), chatSession.Index, lastSeenSignatures, signatureBytes) {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
			return
		}
	}

	broadcastChatMessage(c, data.Message, data.Timestamp, data.Salt, data.Signature, signatureBytes, lastSeenSignatures, isChatMessageSigned)
}

// SendSignedMessage todo: change chat type to accept inline def
func (c *Connection) SendSignedMessage(target *Connection, message string, signature []byte, signed *commander.SignedData, chatType int32) {
	var sig mc.PrefixedOptional[mc.ByteArray, *mc.ByteArray]
	isSigned := len(signature) > 0
	if isSigned {
		sigArray := mc.NewByteArray(256)
		sigArray.Data = signature
		sig = mc.NewPrefixedOptional[mc.ByteArray, *mc.ByteArray](sigArray)
	}

	sendSignedChatPacket(
		c, target,
		mc.String256(message),
		mc.Long(signed.Timestamp), mc.Long(signed.Salt),
		sig, signature, signed.LastSeenSignatures,
		isSigned, chatType,
	)
}

func advanceSession(session *mc.ChatSession, acknowledged *mc.FixedBitSet) [][]byte {
	session.LastSeenCount = 0
	session.Index++
	return getLastSeenSignatures(session, acknowledged)
}

func validateSessionTiming(session *mc.ChatSession, timestampMillis int64) mcdata.TranslationKey {
	now := time.Now()
	if now.UnixMilli() > session.ExpiresAt {
		return mcdata.MultiplayerDisconnectExpiredPublicKey
	}
	messageTime := time.UnixMilli(timestampMillis)
	if messageTime.Before(now.Add(-maxChatTimeSkewPast)) || messageTime.After(now.Add(maxChatTimeSkewFuture)) {
		return mcdata.MultiplayerDisconnectChatValidationFailed
	}
	if session.Index < 0 {
		return mcdata.MultiplayerDisconnectBadChatIndex
	}
	return ""
}

func sendSignedChatPacket(
	sender *Connection,
	target *Connection,
	message mc.String256,
	timestamp, salt mc.Long,
	signature mc.PrefixedOptional[mc.ByteArray, *mc.ByteArray],
	signatureBytes []byte,
	lastSeenSignatures [][]byte,
	isSigned bool,
	chatType int32,
) {
	targetSession := &target.Player.ChatSession
	globalIndex := targetSession.GlobalIndex
	targetSession.GlobalIndex++
	outPkt := sender.NewPacket(
		packet.PlayClientboundPlayerChat,
		mc.VarInt(globalIndex),
		mc.UUID(sender.Player.UUID),
		mc.VarInt(sender.Player.ChatSession.Index),
		signature,
		message,
		timestamp,
		salt,
	)
	if outPkt == nil {
		return
	}

	if isSigned {
		targetSession.LastSeenCount++
		pm := &targetSession.PreviousMessages
		messageID := int32(pm.Len())

		_ = outPkt.Encode(mc.VarInt(len(lastSeenSignatures)))
		for _, sig := range lastSeenSignatures {
			clientMessageID := int32(-1)
			for j := 0; j < pm.Len(); j++ {
				if bytes.Equal(pm.Get(j).Signature, sig) {
					clientMessageID = int32(j)
					break
				}
			}

			_ = outPkt.Encode(mc.VarInt(clientMessageID + 1))
			if clientMessageID == -1 {
				bArray := mc.ByteArray{Data: sig[:256]}
				_ = outPkt.Encode(bArray)
			}
		}
		pm.Add(mc.PreviousMessage{MessageID: messageID, Signature: signatureBytes})
	} else {
		_ = outPkt.Encode(mc.VarInt(0))
	}
	// Unsigned Content is sent when you want to have a styled message (only with no secure chat)
	_ = outPkt.Encode(mc.Boolean(false), mc.VarInt(0), mc.VarInt(chatType), tc.PlayerName(sender.Player.Name), mc.Boolean(false))
	target.Send(outPkt)
}

func broadcastChatMessage(
	sender *Connection,
	message mc.String256,
	timestamp, salt mc.Long,
	signature mc.PrefixedOptional[mc.ByteArray, *mc.ByteArray],
	signatureBytes []byte,
	lastSeenSignatures [][]byte,
	isSigned bool,
) {
	sender.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		if conn.Player == nil {
			return true
		}
		sendSignedChatPacket(sender, conn, message, timestamp, salt, signature, signatureBytes, lastSeenSignatures, isSigned, 1)
		return true
	})
}

func getLastSeenSignatures(session *mc.ChatSession, acknowledged *mc.FixedBitSet) [][]byte {
	lastSeenSigs := make([][]byte, 0, 20)
	n := session.PreviousMessages.Len()
	for j := n - 1; j >= 0; j-- {
		bitIndex := 20 - n + j
		if seen, _ := acknowledged.Get(bitIndex); seen {
			lastSeenSigs = append(lastSeenSigs, session.PreviousMessages.Get(j).Signature)
		}
	}
	return lastSeenSigs
}

func computeLastSeenChecksum(signatures [][]byte) mc.Byte {
	var result int32 = 1
	for _, sig := range signatures {
		sigChecksum := internal.ArrayHash(sig)
		result = 31*result + sigChecksum
	}
	checksum := mc.Byte(byte(result))
	if checksum == 0 {
		return 1
	}
	return checksum
}

func verifyMessageSignature(
	session *mc.ChatSession,
	senderUUID uuid.UUID,
	message string,
	timestampMillis int64,
	salt int64,
	index int32,
	lastSeenSigs [][]byte,
	signature []byte,
) bool {
	buf := verifyBufPool.Get().([]byte)
	defer verifyBufPool.Put(buf[:0])

	buf = binary.BigEndian.AppendUint32(buf, 1)
	buf = append(buf, senderUUID[:]...)
	buf = append(buf, session.ID[:]...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(index))
	buf = binary.BigEndian.AppendUint64(buf, uint64(salt))
	buf = binary.BigEndian.AppendUint64(buf, uint64(timestampMillis/1000))
	msgBytes := []byte(message)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(msgBytes)))
	buf = append(buf, msgBytes...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(lastSeenSigs)))
	for _, sig := range lastSeenSigs {
		buf = append(buf, sig...)
	}

	hash := sha256.Sum256(buf)
	return rsa.VerifyPKCS1v15(session.PublicKey, crypto.SHA256, hash[:], signature) == nil
}

func verifyChatSessionKey(mojangKeys []*rsa.PublicKey, playerUUID uuid.UUID, expiresAt int64, publicKeyBytes []byte, keySignature []byte) error {
	payload := make([]byte, 0, 16+8+len(publicKeyBytes))
	payload = append(payload, playerUUID[:]...)
	payload = binary.BigEndian.AppendUint64(payload, uint64(expiresAt))
	payload = append(payload, publicKeyBytes...)
	hash := sha1.Sum(payload)

	for _, key := range mojangKeys {
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA1, hash[:], keySignature); err == nil {
			return nil
		}
	}
	return fmt.Errorf("key signature could not be verified against any Mojang certificate key")
}

func (c *Connection) playerSource() *commander.CommandSource {
	return &commander.CommandSource{
		PermissionLevel: c.Player.PermissionLevel,
		Server:          c.Server,
		Entity:          c.Player,
		Position:        c.Player.Pos,
		Rotation:        c.Player.Rot,
		SendMessage: func(msg any) {
			if comp, ok := msg.(tc.Component); ok {
				pkt := c.NewPacket(packet.PlayClientboundSystemChat, comp, mc.Boolean(false))
				c.Send(pkt)
			}
		},
	}
}
