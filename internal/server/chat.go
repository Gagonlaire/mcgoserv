package server

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/google/uuid"
)

func (c *Connection) HandleChat(pkt *packet.Packet) {
	var message mc.String
	var timestamp, salt mc.Long
	var signature = mc.NewPrefixedOptional(mc.NewArray[mc.Byte](256))
	var messageCount mc.VarInt
	var acknowledged = mc.NewFixedBitSet(20)
	var checksum mc.Byte

	if err := pkt.Decode(&message, &timestamp, &salt, signature, &messageCount, acknowledged, &checksum); err != nil {
		log.Printf("Error decoding chat packet: %v", err)
		return
	}

	chatSession := c.Player.ChatSession
	if chatSession == nil {
		fmt.Println("no chat session")
		return
	}

	signatureBytes := make([]byte, 256)
	for i, b := range *signature.Value.Slice {
		signatureBytes[i] = byte(b)
	}

	chatSession.ChatIndex++
	lastSeenSigs, err := VerifyChatMessage(chatSession, c.Player.UUID, string(message), int64(timestamp), int64(salt), chatSession.ChatIndex, acknowledged, signatureBytes)
	if err != nil {
		log.Printf("ChatSession verification failed: %v", err)
		return
	}

	senderUUID := mc.UUID(c.Player.UUID)
	c.Server.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		if conn.Player == nil || conn.Player.ChatSession == nil {
			return true
		}

		globalIndex := conn.Player.ChatSession.GlobalIndex
		conn.Player.ChatSession.GlobalIndex++

		pm := &conn.Player.ChatSession.PreviousMessages
		messageID := int32(pm.Len())
		idBySig := make(map[string]int32, pm.Len())
		for j := 0; j < pm.Len(); j++ {
			entry := pm.Get(j)
			idBySig[string(entry.Signature)] = int32(j)
		}
		pm.Add(mc.PreviousMessage{MessageID: messageID, Signature: signatureBytes})

		outPkt, _ := packet.NewPacket(
			packet.PlayClientboundPlayerChat,
			mc.VarInt(globalIndex),
			&senderUUID,
			mc.VarInt(chatSession.ChatIndex),
			mc.Boolean(true),
		)

		outPkt.Buffer.Write(signatureBytes)
		_ = outPkt.Encode(message, timestamp, salt)
		_ = outPkt.Encode(mc.VarInt(len(lastSeenSigs)))
		for _, sig := range lastSeenSigs {
			clientMessageID := int32(-1)
			if id, ok := idBySig[string(sig)]; ok {
				clientMessageID = id
			}
			idPlusOne := clientMessageID + 1
			_ = outPkt.Encode(mc.VarInt(idPlusOne))
			if idPlusOne == 0 {
				bArray := mc.NewArray[mc.Byte](256)
				for i := 0; i < 256; i++ {
					(*bArray.Slice)[i] = mc.Byte(sig[i])
				}
				_ = outPkt.Encode(bArray)
			}
		}

		_ = outPkt.Encode(mc.Boolean(false))
		_ = outPkt.Encode(mc.VarInt(0))
		_ = outPkt.Encode(mc.VarInt(1))
		_ = outPkt.Encode(tc.Text(string(c.Player.Name)))
		_ = outPkt.Encode(mc.Boolean(false))
		conn.Send(outPkt)

		return true
	})
}

func VerifyChatMessage(
	session *mc.ChatSession,
	senderUUID uuid.UUID,
	message string,
	timestampMillis int64,
	salt int64,
	index int32,
	acknowledged *mc.FixedBitSet,
	signature []byte,
) ([][]byte, error) {
	if time.Now().UnixMilli() > session.ExpiresAt {
		return nil, fmt.Errorf("chat session expired")
	}
	timestampSeconds := timestampMillis / 1000

	var lastSeenSigs [][]byte
	n := session.PreviousMessages.Len()
	for j := n - 1; j >= 0; j-- {
		bitIndex := 20 - n + j
		seen, _ := acknowledged.Get(bitIndex)
		if !seen {
			continue
		}
		lastSeenSigs = append(lastSeenSigs, session.PreviousMessages.Get(j).Signature)
	}

	var buf bytes.Buffer

	_ = binary.Write(&buf, binary.BigEndian, int32(1))
	buf.Write(senderUUID[:])
	buf.Write(session.ID[:])
	_ = binary.Write(&buf, binary.BigEndian, index)
	_ = binary.Write(&buf, binary.BigEndian, salt)
	_ = binary.Write(&buf, binary.BigEndian, timestampSeconds)
	msgBytes := []byte(message)
	_ = binary.Write(&buf, binary.BigEndian, int32(len(msgBytes)))
	buf.Write(msgBytes)
	_ = binary.Write(&buf, binary.BigEndian, int32(len(lastSeenSigs)))

	for _, sig := range lastSeenSigs {
		buf.Write(sig)
	}
	hash := sha256.Sum256(buf.Bytes())
	if err := rsa.VerifyPKCS1v15(session.PublicKey, crypto.SHA256, hash[:], signature); err != nil {
		return nil, err
	}

	return lastSeenSigs, nil
}
