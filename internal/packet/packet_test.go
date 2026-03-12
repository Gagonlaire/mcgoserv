package packet

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/google/uuid"
)

func ptr[T any](v T) *T { return &v }

var (
	testPacketID    = mc.VarInt(0x03)
	benchPrimitives = []mc.Field{
		ptr(mc.Boolean(true)),
		ptr(mc.Byte(127)),
		ptr(mc.Short(32000)),
		ptr(mc.Int(2147483647)),
		ptr(mc.Long(9223372036854775807)),
		ptr(mc.Float(3.14159)),
		ptr(mc.Double(2.718281828459)),
		ptr(mc.Position{X: 100, Y: 64, Z: -100}), // Packed 64-bit int
	}
	benchVariable = []mc.Field{
		ptr(mc.VarInt(0)),          // 1 byte fast path
		ptr(mc.VarInt(2147483647)), // 5 byte slow path
		ptr(mc.String("Short")),
		ptr(mc.String("A significantly longer string that forces the buffer to allocate and copy more memory during encoding and decoding.")),
	}
	longSlice    = []mc.Long{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	benchComplex = []mc.Field{
		ptr(mc.UUID(uuid.New())),
		ptr(mc.LpVec3{X: 1.5, Y: 2.5, Z: 3.5}),
		ptr(mc.Slot{Count: 64, ItemID: 1}),
		&mc.PrefixedArray[mc.Long, *mc.Long]{Slice: longSlice},
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
	allFields := append(append(benchPrimitives, benchVariable...), benchComplex...)
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
				p, err := NewPacket(int(testPacketID), fieldsToWriters(allFields)...)
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
			for _, f := range allFields {
				_, _ = f.WriteTo(expectedBuf)
			}

			if p.Len() != expectedBuf.Len() {
				t.Errorf("Payload size mismatch: got %d, want %d", p.Len(), expectedBuf.Len())
			}

			if !bytes.Equal(p.Bytes(), expectedBuf.Bytes()) {
				t.Errorf("Payload content mismatch")
			}
		})
	}
}

func BenchmarkEncode(b *testing.B) {
	benchmarks := []struct {
		name   string
		fields []mc.Field
	}{
		{"Primitives", benchPrimitives},
		{"Variable", benchVariable},
		{"Complex", benchComplex},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			p := &OutboundPacket{
				ID:     testPacketID,
				Buffer: new(bytes.Buffer),
			}
			writers := fieldsToWriters(bm.fields)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				p.Buffer.Reset()
				_ = p.Encode(writers...)
			}
		})
	}
}

func BenchmarkDecode(b *testing.B) {
	benchmarks := []struct {
		name   string
		fields []mc.Field
	}{
		{"Primitives", benchPrimitives},
		{"Variable", benchVariable},
		{"Complex", benchComplex},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			var buf bytes.Buffer
			for _, w := range fieldsToWriters(bm.fields) {
				_, _ = w.WriteTo(&buf)
			}
			encodedBytes := append([]byte(nil), buf.Bytes()...)

			p := &InboundPacket{ID: testPacketID}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				p.data = encodedBytes
				p.reader.Reset(encodedBytes)
				_ = p.Decode(bm.fields...)
			}
		})
	}
}

func BenchmarkNetwork_Send(b *testing.B) {
	p, _ := NewPacket(int(testPacketID), fieldsToWriters(benchPrimitives)...)
	defer p.Free()

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
