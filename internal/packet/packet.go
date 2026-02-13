package packet

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
)

var packetPool = sync.Pool{
	New: func() any {
		return &Packet{
			Buffer: bytes.NewBuffer(make([]byte, 0, 256)),
		}
	},
}

type Packet struct {
	ID       mc.VarInt
	Buffer   *bytes.Buffer
	refCount int32
}

func NewPacket(ID int, fields ...io.WriterTo) (*Packet, error) {
	p := packetPool.Get().(*Packet)
	p.ID = mc.VarInt(ID)
	p.refCount = 1
	p.Buffer.Reset()

	if err := p.Encode(fields...); err != nil {
		p.Free()
		return nil, fmt.Errorf("error encoding packet: %w", err)
	}

	return p, nil
}

func Receive(conn net.Conn) (*Packet, error) {
	var packetLength, packetID mc.VarInt

	if _, err := packetLength.ReadFrom(conn); err != nil {
		return nil, err
	}

	packetData := make([]byte, int(packetLength))
	if _, err := io.ReadFull(conn, packetData); err != nil {
		return nil, err
	}

	n, err := packetID.ReadFrom(bytes.NewBuffer(packetData))
	if err != nil {
		return nil, err
	}

	p := packetPool.Get().(*Packet)
	p.ID = packetID
	p.refCount = 1
	p.Buffer.Reset()
	p.Buffer.Write(packetData[n:])

	return p, nil
}

func (p *Packet) Decode(fields ...mc.Field) error {
	for i, f := range fields {
		if _, err := f.ReadFrom(p.Buffer); err != nil {
			return fmt.Errorf("error decoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *Packet) Encode(fields ...io.WriterTo) error {
	for i, f := range fields {
		if _, err := f.WriteTo(p.Buffer); err != nil {
			return fmt.Errorf("error encoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *Packet) ResetWith(ID int, fields ...io.WriterTo) error {
	p.ID = mc.VarInt(ID)
	p.Buffer.Reset()

	return p.Encode(fields...)
}

func (p *Packet) Send(conn net.Conn) error {
	packetLength := mc.VarInt(p.ID.Len() + p.Buffer.Len())
	buffer := bytes.NewBuffer(make([]byte, 0, int(packetLength)+packetLength.Len()))

	_, _ = packetLength.WriteTo(buffer)
	_, _ = p.ID.WriteTo(buffer)
	_, _ = buffer.Write(p.Buffer.Bytes())
	if _, err := conn.Write(buffer.Bytes()); err != nil {
		return fmt.Errorf("error sending packet: %w", err)
	}

	return nil
}

// Retain increments the reference count for the packet.
// This method must be used when reusing a packet received from an external source
// (such as a router handler) that currently holds ownership of it.
// By calling Retain, you prevent the packet from being freed by the original owner.
func (p *Packet) Retain() {
	atomic.AddInt32(&p.refCount, 1)
}

// Free decrements the reference count for the packet and returns it to the pool if the count reaches zero.
func (p *Packet) Free() {
	newRef := atomic.AddInt32(&p.refCount, -1)

	if newRef < 0 {
		panic(fmt.Sprintf("Packet double free detected! ID: %d", p.ID))
	}
	if newRef == 0 {
		packetPool.Put(p)
	}
}
