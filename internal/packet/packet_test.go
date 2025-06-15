package packet

import (
	"bytes"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"net"
	"testing"
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

func BenchmarkPacket_Receive(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	clientPkt := Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}

	defer serverConn.Close()
	defer clientConn.Close()
	_ = clientPkt.Encode(testPacketFields...)

	go func() {
		for {
			err := clientPkt.Send(clientConn)
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = Receive(serverConn)
	}
}

func BenchmarkPacket_Decode(b *testing.B) {
	pkt := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}
	_ = pkt.Encode(testPacketFields...)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pkt.Buffer.Next(pkt.Buffer.Len())
		_ = pkt.Decode(testPacketFields...)
	}
}

func BenchmarkPacket_Encode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		p := &Packet{
			ID:     testPacketID,
			Buffer: new(bytes.Buffer),
		}

		_ = p.Encode(testPacketFields...)
	}
}

func BenchmarkPacket_Send(b *testing.B) {
	serverConn, clientConn := net.Pipe()
	serverPkt := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}

	defer serverConn.Close()
	defer clientConn.Close()
	_ = serverPkt.Encode(testPacketFields...)

	go func() {
		for {
			_, err := Receive(clientConn)
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = serverPkt.Send(serverConn)
	}
}
