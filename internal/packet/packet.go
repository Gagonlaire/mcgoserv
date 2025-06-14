package packet

import (
	"bytes"
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"io"
	"net"
)

type Packet struct {
	ID     mc.VarInt
	Buffer *bytes.Buffer
}

type Field interface {
	io.WriterTo
	io.ReaderFrom
}

func Receive(conn net.Conn) (*Packet, error) {
	var packetLength, packetID mc.VarInt

	_, err := packetLength.ReadFrom(conn)
	if err != nil {
		return nil, fmt.Errorf("error reading packet length: %w", err)
	}

	n, err := packetID.ReadFrom(conn)
	if err != nil {
		return nil, fmt.Errorf("error reading packet ID: %w", err)
	}

	packetData := make([]byte, int(packetLength)-int(n))
	_, err = io.ReadFull(conn, packetData)
	if err != nil {
		return nil, fmt.Errorf("error reading packet data (expected %d bytes): %w", packetLength, err)
	}

	return &Packet{
		ID:     packetID,
		Buffer: bytes.NewBuffer(packetData),
	}, nil
}

func (p *Packet) Decode(fields ...Field) error {
	for i, f := range fields {
		if _, err := f.ReadFrom(p.Buffer); err != nil {
			return fmt.Errorf("error decoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *Packet) ResetWith(ID mc.VarInt, fields ...Field) error {
	p.ID = ID
	p.Buffer.Reset()

	return p.Encode(fields...)
}

func (p *Packet) Encode(fields ...Field) error {
	for i, f := range fields {
		if _, err := f.WriteTo(p.Buffer); err != nil {
			return fmt.Errorf("error encoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *Packet) Send(conn net.Conn) error {
	packetLength := p.ID.Len() + len(p.Buffer.Bytes())
	buffer := bytes.NewBuffer(make([]byte, 0, packetLength+mc.VarInt(packetLength).Len()))

	_, _ = mc.VarInt(packetLength).WriteTo(buffer)
	_, _ = p.ID.WriteTo(buffer)
	_, _ = buffer.Write(p.Buffer.Bytes())

	_, err := conn.Write(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("error sending packet: %w", err)
	}

	return nil
}
