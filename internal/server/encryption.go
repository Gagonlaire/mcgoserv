package server

import (
	"crypto/aes"
	"crypto/cipher"
	"net"
)

type EncryptedConn struct {
	net.Conn
	encrypter cipher.Stream
	decrypter cipher.Stream
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
	buf := make([]byte, len(p))
	ec.encrypter.XORKeyStream(buf, p)
	return ec.Conn.Write(buf)
}

type cfb8Stream struct {
	block   cipher.Block
	iv      []byte
	tmp     []byte
	decrypt bool
}

func newCFB8Stream(block cipher.Block, iv []byte, decrypt bool) cipher.Stream {
	return &cfb8Stream{
		block:   block,
		iv:      iv,
		tmp:     make([]byte, block.BlockSize()),
		decrypt: decrypt,
	}
}

func (s *cfb8Stream) XORKeyStream(dst, src []byte) {
	blockSize := s.block.BlockSize()

	for i := range src {
		s.block.Encrypt(s.tmp, s.iv)
		val := src[i] ^ s.tmp[0]
		dst[i] = val
		copy(s.iv, s.iv[1:])

		if s.decrypt {
			s.iv[blockSize-1] = val ^ s.tmp[0]
		} else {
			s.iv[blockSize-1] = val
		}
	}
}
