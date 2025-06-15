package packet

import (
	"bytes"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"io"
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
	defer serverConn.Close()
	defer clientConn.Close()

	clientPkt := Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}

	_ = clientPkt.Encode(testPacketFields...)
	packetLen := mc.VarInt(clientPkt.ID.Len() + clientPkt.Buffer.Len())
	buf := bytes.NewBuffer(make([]byte, 0, packetLen.Len()+packetLen.Len()))
	_, _ = packetLen.WriteTo(buf)
	_, _ = clientPkt.ID.WriteTo(buf)
	_, _ = clientPkt.Buffer.WriteTo(buf)

	go func() {

		for {
			_, err := clientConn.Write(buf.Bytes())
			if err != nil {
				return
			}
		}
	}()

	b.ReportAllocs()
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

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		pkt.Buffer.Next(pkt.Buffer.Len())
		_ = pkt.Decode(testPacketFields...)
	}
}

func BenchmarkPacket_Encode(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

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
	defer serverConn.Close()
	defer clientConn.Close()

	go func() {
		_, _ = io.Copy(io.Discard, clientConn)
	}()

	templatePacket := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}
	_ = templatePacket.Encode(testPacketFields...)
	initialEncodedData := templatePacket.Buffer.Bytes()
	packetToSend := &Packet{
		ID:     testPacketID,
		Buffer: new(bytes.Buffer),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		packetToSend.Buffer.Reset()
		packetToSend.Buffer.Write(initialEncodedData)

		_ = packetToSend.Send(serverConn)
	}
}
