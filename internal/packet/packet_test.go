package packet

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

var (
	field1           = mc.String("Hello, World!")
	field2           = mc.VarInt(100)
	field3           = mc.Long(200)
	field4           = mc.String("Another string field")
	field5           = mc.UnsignedShort(10)
	testPacketID     = mc.VarInt(0x03)
	testPacketFields = []mc.Field{
		&field1,
		&field2,
		&field3,
		&field4,
		&field5,
	}
)

func fieldsToWriters(fields []mc.Field) []io.WriterTo {
	out := make([]io.WriterTo, len(fields))
	for i, f := range fields {
		out[i] = f
	}
	return out
}

func TestPacket_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
	}{
		{"NoCompression", -1},
		{"CompressionEnabled_BelowThreshold", 10000},
		{"CompressionEnabled_AboveThreshold", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConn, clientConn := net.Pipe()
			defer serverConn.Close()
			defer clientConn.Close()

			go func() {
				p, err := NewPacket(int(testPacketID), fieldsToWriters(testPacketFields)...)
				if err != nil {
					t.Errorf("NewPacket error: %v", err)
					return
				}
				defer p.Free()

				if err := p.Send(clientConn, tt.threshold); err != nil {
					t.Errorf("Send error: %v", err)
				}
			}()

			p, err := Receive(serverConn, tt.threshold)
			if err != nil {
				t.Fatalf("Receive error: %v", err)
			}
			defer p.Free()

			if p.ID != testPacketID {
				t.Errorf("ID mismatch: got %d, want %d", p.ID, testPacketID)
			}

			expectedBuf := new(bytes.Buffer)
			for _, f := range testPacketFields {
				_, _ = f.WriteTo(expectedBuf)
			}

			if p.Buffer.Len() != expectedBuf.Len() {
				t.Errorf("Payload size mismatch: got %d, want %d", p.Buffer.Len(), expectedBuf.Len())
			}

			if !bytes.Equal(p.Buffer.Bytes(), expectedBuf.Bytes()) {
				t.Errorf("Payload content mismatch")
			}
		})
	}
}

func BenchmarkPacket_Receive(b *testing.B) {
	benchmarks := []struct {
		name      string
		threshold int
	}{
		{"Uncompressed", -1},
		{"Compressed", 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			serverConn, clientConn := net.Pipe()
			defer serverConn.Close()
			defer clientConn.Close()

			genIn, genOut := net.Pipe()
			wireDataBuf := new(bytes.Buffer)
			go func() {
				p, _ := NewPacket(int(testPacketID), fieldsToWriters(testPacketFields)...)
				defer p.Free()
				_ = p.Send(genIn, bm.threshold)
				_ = genIn.Close()
			}()
			_, _ = io.Copy(wireDataBuf, genOut)
			wireBytes := wireDataBuf.Bytes()

			go func() {
				for {
					if _, err := clientConn.Write(wireBytes); err != nil {
						return
					}
				}
			}()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				p, err := Receive(serverConn, bm.threshold)
				if err != nil {
					b.Fatalf("Receive failed: %v", err)
				}
				p.Free()
			}
		})
	}
}

func BenchmarkPacket_Send(b *testing.B) {
	benchmarks := []struct {
		name      string
		threshold int
	}{
		{"Uncompressed", -1},
		{"Compressed", 10},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			serverConn, clientConn := net.Pipe()
			defer serverConn.Close()
			defer clientConn.Close()

			go func() {
				_, _ = io.Copy(io.Discard, clientConn)
			}()

			p, _ := NewPacket(int(testPacketID), fieldsToWriters(testPacketFields)...)
			defer p.Free()

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if err := p.Send(serverConn, bm.threshold); err != nil {
					b.Fatalf("Send failed: %v", err)
				}
			}
		})
	}
}

func BenchmarkPacket_Decode(b *testing.B) {
	pkt := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}
	_ = pkt.Encode(fieldsToWriters(testPacketFields)...)
	encodedBytes := pkt.Buffer.Bytes()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pkt.Buffer = bytes.NewBuffer(encodedBytes)
		_ = pkt.Decode(testPacketFields...)
	}
}

func BenchmarkPacket_Encode(b *testing.B) {
	p := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}

	writers := fieldsToWriters(testPacketFields)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p.Buffer.Reset()
		_ = p.Encode(writers...)
	}
}
