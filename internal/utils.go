package internal

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

const (
	AnsiReset     = "\u001B[0m"
	AnsiBold      = "\u001B[1m"
	AnsiItalic    = "\u001B[3m"
	AnsiUnderline = "\u001B[4m"
	AnsiStrike    = "\u001B[9m"
	ColorRed      = "\u001B[31m"
	ColorGreen    = "\u001B[32m"
	ColorYellow   = "\u001B[33m"
	ColorBlue     = "\u001B[34m"
	ColorPurple   = "\u001B[35m"
	ColorCyan     = "\u001B[36m"
	ColorWhite    = "\u001B[37m"
)

// AuthDigest https://minecraft.wiki/w/Java_Edition_protocol/Encryption#Sample_Code
func AuthDigest(s string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, s)
	hash := h.Sum(nil)
	negative := (hash[0] & 0x80) == 0x80
	if negative {
		hash = twosComplement(hash)
	}

	res := strings.TrimLeft(hex.EncodeToString(hash), "0")
	if negative {
		res = "-" + res
	}

	return res
}

func twosComplement(p []byte) []byte {
	carry := true
	for i := len(p) - 1; i >= 0; i-- {
		p[i] = ^p[i]
		if carry {
			carry = p[i] == 0xff
			p[i]++
		}
	}
	return p
}

func EqualBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func GetOfflineUUID(name string) uuid.UUID {
	h := md5.New()
	h.Write([]byte("OfflinePlayer:" + name))
	digest := h.Sum(nil)
	digest[6] = (digest[6] & 0x0f) | 0x30
	digest[8] = (digest[8] & 0x3f) | 0x80
	var u uuid.UUID
	copy(u[:], digest)
	return u
}

func VerifyChatSessionKey(keys []*rsa.PublicKey, playerUUID uuid.UUID, expiresAt int64, publicKeyBytes []byte, keySignature []byte) error {
	payload := make([]byte, 0, 16+8+len(publicKeyBytes))
	payload = append(payload, playerUUID[:]...)
	payload = binary.BigEndian.AppendUint64(payload, uint64(expiresAt))
	payload = append(payload, publicKeyBytes...)
	hash := sha1.Sum(payload)

	for _, key := range keys {
		if err := rsa.VerifyPKCS1v15(key, crypto.SHA1, hash[:], keySignature); err == nil {
			return nil
		}
	}
	return fmt.Errorf("key signature could not be verified against any Mojang certificate key")
}
