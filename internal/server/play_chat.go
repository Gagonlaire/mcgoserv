package server

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
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
		resp, _ := packet.NewPacket(
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
	resp, _ := packet.NewPacket(
		packet.PlayClientboundCommandSuggestions,
		data.TransactionID,
		mc.VarInt(startIndex),
		mc.VarInt(ctx.Length),
		mc.VarInt(len(entries)),
	)
	for _, entry := range entries {
		_ = resp.Encode(mc.String(entry.Text), mc.Boolean(entry.Tooltip != nil))
		if entry.Tooltip != nil {
			_ = resp.Encode(entry.Tooltip)
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

	publicKeyBytes := mc.MapToSlice(data.PublicKey, func(b mc.Byte) byte { return byte(b) })
	signatureBytes := mc.MapToSlice(data.KeySignature, func(b mc.Byte) byte { return byte(b) })

	if err := VerifyChatSessionKey(
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
	pkt, _ := buildPlayerInfoUpdatePacket(mc.ActionInitializeChat, player)
	c.Server.Broadcaster.Broadcast(pkt)
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
	// todo: create a parse method that stores raw arguments for signature verification
	if c.Server.EnforceSecureChat {
		// todo: validate signatures
	} else {

	}

	src := c.playerSource()
	// todo: commands should maybe ran in a separate routine
	_, err := c.Server.Commander.ExecuteInput(
		c.ctx,
		src,
		string(data.Command),
	)

	if err != nil {
		src.SendMessage(commander.AsCommandError(err).ToComponent())
	}
}

func (c *Connection) HandleChatMessage(data *decoders.ChatMessage) {
	chatSession := &c.Player.ChatSession
	isChatMessageSigned := chatSession.Signed && bool(data.Signature.Has)
	if c.Server.EnforceSecureChat {
		if !isChatMessageSigned {
			unsignedErrorPkt, _ := packet.NewPacket(
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

	chatSession.LastSeenCount = 0
	chatSession.Index++
	lastSeenSignatures := getLastSeenSignatures(chatSession, data.Acknowledged)
	expectedChecksum := computeLastSeenChecksum(lastSeenSignatures)
	if data.Checksum != expectedChecksum {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
		return
	}

	var signatureBytes []byte
	if isChatMessageSigned {
		signatureBytes = make([]byte, 256)
		for i, b := range data.Signature.Value.Slice {
			signatureBytes[i] = byte(b)
		}

		err, ok := verifyChatMessage(chatSession, c.Player.UUID, string(data.Message), int64(data.Timestamp), int64(data.Salt), chatSession.Index, lastSeenSignatures, signatureBytes)
		if !ok {
			c.Disconnect(tc.Translatable(err))
		}
	}

	broadcastChatMessage(c, data.Message, data.Timestamp, data.Salt, data.Signature, signatureBytes, lastSeenSignatures, isChatMessageSigned)
}

func broadcastChatMessage(
	sender *Connection,
	message mc.String256,
	timestamp, salt mc.Long,
	signature mc.PrefixedOptional[mc.Array[mc.Byte, *mc.Byte], *mc.Array[mc.Byte, *mc.Byte]],
	signatureBytes []byte,
	lastSeenSignatures [][]byte,
	isSigned bool,
) {
	senderUUID := mc.UUID(sender.Player.UUID)
	sender.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		if conn.Player == nil {
			return true
		}

		globalIndex := conn.Player.ChatSession.GlobalIndex
		conn.Player.ChatSession.GlobalIndex++
		outPkt, _ := packet.NewPacket(
			packet.PlayClientboundPlayerChat,
			mc.VarInt(globalIndex),
			&senderUUID,
			mc.VarInt(sender.Player.ChatSession.Index),
			signature,
			message,
			timestamp,
			salt,
		)

		if isSigned {
			conn.Player.ChatSession.LastSeenCount++
			pm := &conn.Player.ChatSession.PreviousMessages
			messageID := int32(pm.Len())

			_ = outPkt.Encode(mc.VarInt(len(lastSeenSignatures)))
			for _, sig := range lastSeenSignatures {
				clientMessageID := int32(-1)
				for j := 0; j < pm.Len(); j++ {
					if bytes.Equal(pm.Get(j).Signature, sig) {
						clientMessageID = int32(j)
					}
				}

				_ = outPkt.Encode(mc.VarInt(clientMessageID + 1))
				if clientMessageID == -1 {
					bArray := mc.NewArray[mc.Byte, *mc.Byte](256)
					for i := 0; i < 256; i++ {
						bArray.Slice[i] = mc.Byte(sig[i])
					}
					_ = outPkt.Encode(bArray)
				}
			}
			pm.Add(mc.PreviousMessage{MessageID: messageID, Signature: signatureBytes})
		} else {
			_ = outPkt.Encode(mc.VarInt(0))
		}
		// Unsigned Content is send when you want to have a styled message (only with no secure chat)
		_ = outPkt.Encode(mc.Boolean(false), mc.VarInt(0), mc.VarInt(1), tc.PlayerName(sender.Player.Name), mc.Boolean(false))
		conn.Send(outPkt)

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

func verifyChatMessage(
	session *mc.ChatSession,
	senderUUID uuid.UUID,
	message string,
	timestampMillis int64,
	salt int64,
	index int32,
	lastSeenSigs [][]byte,
	signature []byte,
) (mcdata.TranslationKey, bool) {
	now := time.Now()
	messageTime := time.UnixMilli(timestampMillis)

	if now.UnixMilli() > session.ExpiresAt {
		return mcdata.MultiplayerDisconnectExpiredPublicKey, false
	}
	if messageTime.Before(now.Add(-maxChatTimeSkewPast)) || messageTime.After(now.Add(maxChatTimeSkewFuture)) {
		return mcdata.MultiplayerDisconnectChatValidationFailed, false
	}
	if index < 0 {
		return mcdata.MultiplayerDisconnectBadChatIndex, false
	}

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
	if err := rsa.VerifyPKCS1v15(session.PublicKey, crypto.SHA256, hash[:], signature); err != nil {
		return mcdata.MultiplayerDisconnectChatValidationFailed, false
	}

	return "", true
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
				pkt, _ := packet.NewPacket(packet.PlayClientboundSystemChat, comp, mc.Boolean(false))
				c.Send(pkt)
			}
		},
	}
}
