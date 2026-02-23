package packet

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/klauspost/compress/zlib"
)

type Packet struct {
	ID       mc.VarInt
	Buffer   *bytes.Buffer
	refCount int32
}

// MaxUncompressedSize https://minecraft.wiki/w/Java_Edition_protocol/Packets#With_compression
const (
	MaxUncompressedSize = 8388608
	MaxFrameSize        = MaxUncompressedSize + 512
	MaxPooledBufferCap  = 65536
)

var packetPool = sync.Pool{
	New: func() any {
		return &Packet{
			Buffer: bytes.NewBuffer(make([]byte, 0, 256)),
		}
	},
}

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 1024))
	},
}

var emptyZlibPayload = func() []byte {
	var b bytes.Buffer
	writer, _ := zlib.NewWriterLevel(&b, zlib.NoCompression)

	writer.Close()
	return b.Bytes()
}()

var zlibReaderPool = sync.Pool{
	New: func() any {
		reader, _ := zlib.NewReader(bytes.NewReader(emptyZlibPayload))
		return reader
	},
}

var zlibWriterPool = sync.Pool{
	New: func() any {
		writer, _ := zlib.NewWriterLevel(nil, zlib.DefaultCompression)
		return writer
	},
}

func getZlibReader(r io.Reader) io.ReadCloser {
	reader := zlibReaderPool.Get().(io.ReadCloser)

	// check why zlib.Reaser is not exported
	if readerConcrete, ok := reader.(interface{ Reset(io.Reader) error }); ok {
		_ = readerConcrete.Reset(r)
	} else {
		_ = reader.Close()
		reader, _ = zlib.NewReader(r)
	}
	return reader
}

func getZlibWriter(w io.Writer) *zlib.Writer {
	writer := zlibWriterPool.Get().(*zlib.Writer)

	writer.Reset(w)
	return writer
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

func Receive(conn net.Conn, threshold int) (*Packet, error) {
	var frameLength mc.VarInt
	if _, err := frameLength.ReadFrom(conn); err != nil {
		return nil, fmt.Errorf("read frame length: %w", err)
	}

	if int(frameLength) > MaxFrameSize || frameLength < 0 {
		return nil, fmt.Errorf("invalid frame length: %d", frameLength)
	}

	rawBuf := bufferPool.Get().(*bytes.Buffer)
	rawBuf.Reset()

	defer func() {
		if rawBuf.Cap() > MaxPooledBufferCap {
			return
		}
		bufferPool.Put(rawBuf)
	}()

	if _, err := io.CopyN(rawBuf, conn, int64(frameLength)); err != nil {
		return nil, fmt.Errorf("read frame body: %w", err)
	}

	var bodyReader io.Reader = rawBuf
	if threshold >= 0 {
		var dataLength mc.VarInt
		if _, err := dataLength.ReadFrom(bodyReader); err != nil {
			return nil, fmt.Errorf("read data length: %w", err)
		}

		if dataLength != 0 {
			if int(dataLength) > MaxUncompressedSize {
				return nil, fmt.Errorf("uncompressed size %d > limit", dataLength)
			}
			if int(dataLength) < threshold {
				return nil, fmt.Errorf("compressed packet size %d < threshold %d", dataLength, threshold)
			}

			zlibReader := getZlibReader(bodyReader)
			defer func() {
				_ = zlibReader.Close()
				zlibReaderPool.Put(zlibReader)
			}()
			bodyReader = zlibReader
		}
	}

	var packetID mc.VarInt
	if _, err := packetID.ReadFrom(bodyReader); err != nil {
		return nil, fmt.Errorf("read packet ID: %w", err)
	}

	p := packetPool.Get().(*Packet)
	p.ID = packetID
	p.refCount = 1
	p.Buffer.Reset()
	if _, err := p.Buffer.ReadFrom(bodyReader); err != nil {
		p.Free()
		return nil, fmt.Errorf("read packet content: %w", err)
	}

	return p, nil
}

func (p *Packet) Send(conn net.Conn, threshold int) error {
	// todo: add zero-copy optimization for uncompressed packets and packet batching
	frame := bufferPool.Get().(*bytes.Buffer)
	frame.Reset()
	defer bufferPool.Put(frame)

	uncompressedSize := p.ID.Len() + p.Buffer.Len()
	if uncompressedSize > MaxUncompressedSize {
		return fmt.Errorf("packet size %d exceeds protocol limit", uncompressedSize)
	}

	if threshold >= 0 && uncompressedSize >= threshold {
		dataLength := mc.VarInt(uncompressedSize)
		compBuf := bufferPool.Get().(*bytes.Buffer)
		compBuf.Reset()
		defer bufferPool.Put(compBuf)

		writer := getZlibWriter(compBuf)

		_, _ = p.ID.WriteTo(writer)
		_, _ = p.Buffer.WriteTo(writer)
		_ = writer.Close()
		zlibWriterPool.Put(writer)
		packetLength := mc.VarInt(dataLength.Len() + compBuf.Len())
		_, _ = packetLength.WriteTo(frame)
		_, _ = dataLength.WriteTo(frame)
		_, _ = compBuf.WriteTo(frame)
	} else {
		packetLengthValue := uncompressedSize
		if threshold >= 0 {
			buf := mc.VarInt(0)
			packetLengthValue += buf.Len()
		}

		packetLength := mc.VarInt(packetLengthValue)
		_, _ = packetLength.WriteTo(frame)

		if threshold >= 0 {
			_, _ = mc.VarInt(0).WriteTo(frame)
		}

		_, _ = p.ID.WriteTo(frame)
		_, _ = p.Buffer.WriteTo(frame)
	}

	_, err := frame.WriteTo(conn)
	return err
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

// Retain increments the reference count for the packet.
// This method must be used when reusing a packet received from an external source
// (such as a router handler) that currently holds ownership of it.
// By calling Retain, you prevent the packet from being freed by the original owner.
func (p *Packet) Retain() {
	if atomic.LoadInt32(&p.refCount) <= 0 {
		panic(fmt.Sprintf("Retain on freed packet detected! ID: %d", p.ID))
	}
	atomic.AddInt32(&p.refCount, 1)
}

// Free decrements the reference count for the packet and returns it to the pool if the count reaches zero.
func (p *Packet) Free() {
	newRef := atomic.AddInt32(&p.refCount, -1)

	switch {
	case newRef < 0:
		panic(fmt.Sprintf("Packet double free detected! ID: %d", p.ID))
	case newRef == 0:
		if p.Buffer.Cap() > MaxPooledBufferCap {
			p.Buffer = bytes.NewBuffer(make([]byte, 0, 128))
		} else {
			p.Buffer.Reset()
		}
		p.ID = 0
		packetPool.Put(p)
	}
}
