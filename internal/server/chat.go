package server

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
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

func (c *Connection) HandleChat(pkt *packet.Packet) {
	var message mc.String
	var timestamp, salt mc.Long
	var signature = mc.NewPrefixedOptional(mc.NewArray[mc.Byte](256))
	var messageCount mc.VarInt
	var acknowledged = mc.NewFixedBitSet(20)
	var checksum mc.Byte

	if err := pkt.Decode(&message, &timestamp, &salt, signature, &messageCount, acknowledged, &checksum); err != nil {
		c.Disconnect(tc.Translatable(mcdata.DisconnectPacketError))
		return
	}

	chatSession := &c.Player.ChatSession
	isChatMessageSigned := chatSession.Signed && bool(signature.Has)
	if c.Server.EnforceSecureChat {
		if !isChatMessageSigned {
			unsignedErrorPkt, _ := packet.NewPacket(
				packet.PlayClientboundSystemChat,
				tc.Text("The server refused to deliver an unsigned message. You can enable chat signing by changing your signing mode or through a prompt screen").SetColor(tc.ColorRed),
				mc.Boolean(false),
			)
			c.Send(unsignedErrorPkt)
			return
		} else if int32(messageCount) > chatSession.LastSeenCount {
			c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectBadChatIndex))
			return
		}
	}

	chatSession.LastSeenCount = 0
	chatSession.Index++
	lastSeenSignatures := GetLastSeenSignatures(chatSession, acknowledged)
	expectedChecksum := ComputeLastSeenChecksum(lastSeenSignatures)
	if checksum != expectedChecksum {
		c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectChatValidationFailed))
		return
	}

	var signatureBytes []byte
	if isChatMessageSigned {
		signatureBytes = make([]byte, 256)
		for i, b := range *signature.Value.Slice {
			signatureBytes[i] = byte(b)
		}

		err, ok := VerifyChatMessage(chatSession, c.Player.UUID, string(message), int64(timestamp), int64(salt), chatSession.Index, lastSeenSignatures, signatureBytes)
		if !ok {
			c.Disconnect(tc.Translatable(err))
		}
	}

	BroadcastChatMessage(c, message, timestamp, salt, signature, signatureBytes, lastSeenSignatures, isChatMessageSigned)
}

func BroadcastChatMessage(
	sender *Connection,
	message mc.String,
	timestamp, salt mc.Long,
	signature *mc.PrefixedOptional[mc.Array[mc.Byte]],
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
					bArray := mc.NewArray[mc.Byte](256)
					for i := 0; i < 256; i++ {
						(*bArray.Slice)[i] = mc.Byte(sig[i])
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

func GetLastSeenSignatures(session *mc.ChatSession, acknowledged *mc.FixedBitSet) [][]byte {
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

func ComputeLastSeenChecksum(signatures [][]byte) mc.Byte {
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

func VerifyChatMessage(
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
