package server

import (
	"bytes"
	"crypto/aes"
	"net"
	"sync"
	"testing"
	"time"
)

func runConcurrentWrites(t *testing.T, lock *sync.Mutex) []byte {
	t.Helper()
	secret := bytes.Repeat([]byte{0xAB}, 16)

	serverSide, clientSide := net.Pipe()
	defer serverSide.Close()
	defer clientSide.Close()

	enc, err := NewEncryptedConn(serverSide, secret)
	if err != nil {
		t.Fatalf("NewEncryptedConn: %v", err)
	}

	block, _ := aes.NewCipher(secret)
	iv := make([]byte, len(secret))
	copy(iv, secret)
	clientDec := newCFB8Stream(block, iv, true)

	const writers = 8
	const writesPer = 200
	const payloadLen = 64
	totalBytes := writers * writesPer * payloadLen

	received := make(chan []byte, 1)
	go func() {
		buf := make([]byte, totalBytes)
		_ = clientSide.SetReadDeadline(time.Now().Add(5 * time.Second))
		n := 0
		for n < totalBytes {
			read, err := clientSide.Read(buf[n:])
			if err != nil {
				break
			}
			n += read
		}
		clientDec.XORKeyStream(buf[:n], buf[:n])
		received <- buf[:n]
	}()

	var wg sync.WaitGroup
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for k := 0; k < writesPer; k++ {
				p := make([]byte, payloadLen)
				for j := range p {
					p[j] = byte(idx + 1)
				}
				if lock != nil {
					lock.Lock()
				}
				_, _ = enc.WriteInPlace(p)
				if lock != nil {
					lock.Unlock()
				}
			}
		}(i)
	}
	wg.Wait()
	return <-received
}

// TestEncryptedConn_SerializedWritesRoundTrip is the regression test for mcg-10-bug-packet-error-on-login
func TestEncryptedConn_SerializedWritesRoundTrip(t *testing.T) {
	var mu sync.Mutex
	got := runConcurrentWrites(t, &mu)

	const writers = 8
	const writesPer = 200
	const payloadLen = 64

	counts := make(map[byte]int)
	for _, b := range got {
		counts[b]++
	}
	for i := 0; i < writers; i++ {
		want := writesPer * payloadLen
		if counts[byte(i+1)] != want {
			t.Errorf("byte 0x%02x: got %d, want %d", i+1, counts[byte(i+1)], want)
		}
	}
}
