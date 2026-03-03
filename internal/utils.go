package internal

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
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

// ArrayHash same logic as Java's Arrays.hashCode(byte[]) implementation.
func ArrayHash(data []byte) int32 {
	var result int32 = 1
	for _, b := range data {
		result = 31*result + int32(int8(b))
	}
	return result
}
