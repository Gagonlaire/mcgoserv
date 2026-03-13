package server

import (
	"crypto/aes"
	"crypto/cipher"
	"net"
)

type InPlaceWriter interface {
	WriteInPlace(p []byte) (int, error)
}

type EncryptedConn struct {
	net.Conn
	encrypter cipher.Stream
	decrypter cipher.Stream
	writeBuf  []byte
}

func NewEncryptedConn(conn net.Conn, sharedSecret []byte) (*EncryptedConn, error) {
	block, err := aes.NewCipher(sharedSecret)
	if err != nil {
		return nil, err
	}

	iv := make([]byte, len(sharedSecret))
	copy(iv, sharedSecret)
	encIV := make([]byte, len(sharedSecret))
	decIV := make([]byte, len(sharedSecret))
	copy(encIV, iv)
	copy(decIV, iv)

	return &EncryptedConn{
		Conn:      conn,
		encrypter: newCFB8Stream(block, encIV, false),
		decrypter: newCFB8Stream(block, decIV, true),
		writeBuf:  make([]byte, 4096),
	}, nil
}

func (ec *EncryptedConn) Read(p []byte) (int, error) {
	n, err := ec.Conn.Read(p)
	if n > 0 {
		ec.decrypter.XORKeyStream(p[:n], p[:n])
	}
	return n, err
}

func (ec *EncryptedConn) Write(p []byte) (int, error) {
	if len(p) > cap(ec.writeBuf) {
		ec.writeBuf = make([]byte, len(p))
	}
	dst := ec.writeBuf[:len(p)]
	copy(dst, p)
	ec.encrypter.XORKeyStream(dst, dst)
	return ec.Conn.Write(dst)
}

func (ec *EncryptedConn) WriteInPlace(p []byte) (int, error) {
	ec.encrypter.XORKeyStream(p, p)
	return ec.Conn.Write(p)
}

type cfb8Stream struct {
	block   cipher.Block
	ring    [32]byte // ring buffer: double blockSize to avoid per-byte copy
	tmp     [16]byte
	pos     int
	decrypt bool
}

func newCFB8Stream(block cipher.Block, iv []byte, decrypt bool) cipher.Stream {
	s := &cfb8Stream{
		decrypt: decrypt,
		block:   block,
	}
	copy(s.ring[:], iv)
	return s
}

func (s *cfb8Stream) XORKeyStream(dst, src []byte) {
	blockSize := s.block.BlockSize()

	for i := range src {
		s.block.Encrypt(s.tmp[:], s.ring[s.pos:s.pos+blockSize])
		val := src[i] ^ s.tmp[0]
		dst[i] = val

		if s.decrypt {
			s.ring[s.pos+blockSize] = val ^ s.tmp[0]
		} else {
			s.ring[s.pos+blockSize] = val
		}
		s.pos++

		if s.pos == blockSize {
			copy(s.ring[:blockSize], s.ring[blockSize:2*blockSize])
			s.pos = 0
		}
	}
}
