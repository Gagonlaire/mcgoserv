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

// MaxUncompressedSize https://minecraft.wiki/w/Java_Edition_protocol/Packets#With_compression
const (
	MaxUncompressedSize = 8388608
	MaxFrameSize        = MaxUncompressedSize + 512
	MaxPooledBufferCap  = 65536
)

type InboundPacket struct {
	data     []byte
	reader   bytes.Reader
	ID       mc.VarInt
	refCount int32
}

type OutboundPacket struct {
	Buffer   *bytes.Buffer
	ID       mc.VarInt
	refCount int32
}

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, 4096))
	},
}

var inboundDataPool = sync.Pool{
	New: func() any { return make([]byte, 0, 4096) },
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

func NewPacket(ID int, fields ...io.WriterTo) (*OutboundPacket, error) {
	p := &OutboundPacket{
		ID:       mc.VarInt(ID),
		Buffer:   bufferPool.Get().(*bytes.Buffer),
		refCount: 1,
	}
	p.Buffer.Reset()
	if err := p.Encode(fields...); err != nil {
		p.Free()
		return nil, fmt.Errorf("error encoding packet: %w", err)
	}
	return p, nil
}

func Receive(conn net.Conn, threshold int) (*InboundPacket, error) {
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
		if rawBuf.Cap() <= MaxPooledBufferCap {
			bufferPool.Put(rawBuf)
		}
	}()
	if _, err := io.CopyN(rawBuf, conn, int64(frameLength)); err != nil {
		return nil, fmt.Errorf("read frame body: %w", err)
	}
	var bodyBuf = rawBuf

	if threshold >= 0 {
		var dataLength mc.VarInt
		if _, err := dataLength.ReadFrom(rawBuf); err != nil {
			return nil, fmt.Errorf("read data length: %w", err)
		}

		if dataLength != 0 {
			if int(dataLength) > MaxUncompressedSize {
				return nil, fmt.Errorf("uncompressed size %d > limit", dataLength)
			}
			if int(dataLength) < threshold {
				return nil, fmt.Errorf("compressed size %d < threshold %d", dataLength, threshold)
			}

			decompBuf := bufferPool.Get().(*bytes.Buffer)
			decompBuf.Reset()
			defer func() {
				if decompBuf.Cap() <= MaxPooledBufferCap {
					bufferPool.Put(decompBuf)
				}
			}()

			zlibReader := getZlibReader(rawBuf)
			defer func() {
				_ = zlibReader.Close()
				zlibReaderPool.Put(zlibReader)
			}()

			if _, err := io.Copy(decompBuf, zlibReader); err != nil {
				return nil, fmt.Errorf("decompress packet: %w", err)
			}
			bodyBuf = decompBuf
		}
	}

	var packetID mc.VarInt
	if _, err := packetID.ReadFrom(bodyBuf); err != nil {
		return nil, fmt.Errorf("read packet ID: %w", err)
	}

	p := &InboundPacket{
		ID:       packetID,
		refCount: 1,
	}
	data := inboundDataPool.Get().([]byte)
	p.data = append(data[:0], bodyBuf.Bytes()...)
	p.reader.Reset(p.data)

	return p, nil
}

func (p *OutboundPacket) Send(conn net.Conn, threshold int) error {
	return writeFramedPacket(conn, p.ID, p.Buffer.Bytes(), threshold)
}

// Forward is only used by packet sniffer
func (p *InboundPacket) Forward(conn net.Conn, threshold int) error {
	return writeFramedPacket(conn, p.ID, p.data, threshold)
}

func (p *InboundPacket) Decode(fields ...mc.Field) error {
	for i, f := range fields {
		if _, err := f.ReadFrom(&p.reader); err != nil {
			return fmt.Errorf("error decoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *OutboundPacket) Encode(fields ...io.WriterTo) error {
	for i, f := range fields {
		if _, err := f.WriteTo(p.Buffer); err != nil {
			return fmt.Errorf("error encoding field %d: %w", i, err)
		}
	}
	return nil
}

func (p *OutboundPacket) ResetWith(ID int, fields ...io.WriterTo) error {
	p.ID = mc.VarInt(ID)
	p.Buffer.Reset()

	return p.Encode(fields...)
}

func (p *InboundPacket) Bytes() []byte { return p.data }

func (p *InboundPacket) Len() int { return len(p.data) }

// Retain increments the reference count for the packet.
func (p *InboundPacket) Retain() {
	if atomic.LoadInt32(&p.refCount) <= 0 {
		panic(fmt.Sprintf("Retain on freed packet detected! ID: %d", p.ID))
	}
	atomic.AddInt32(&p.refCount, 1)
}

// Free decrements the reference count for the packet and returns it to the pool if the count reaches zero.
func (p *InboundPacket) Free() {
	newRef := atomic.AddInt32(&p.refCount, -1)

	switch {
	case newRef < 0:
		panic(fmt.Sprintf("Packet double free detected! ID: %d", p.ID))
	case newRef == 0:
		if cap(p.data) <= MaxPooledBufferCap {
			inboundDataPool.Put(p.data[:0])
		}
		p.data = nil
		p.reader.Reset(nil)
		p.ID = 0
	}
}

// Retain increments the reference count for the packet.
func (p *OutboundPacket) Retain() {
	if atomic.LoadInt32(&p.refCount) <= 0 {
		panic(fmt.Sprintf("Retain on freed packet detected! ID: %d", p.ID))
	}
	atomic.AddInt32(&p.refCount, 1)
}

// Free decrements the reference count for the packet and returns it to the pool if the count reaches zero.
func (p *OutboundPacket) Free() {
	newRef := atomic.AddInt32(&p.refCount, -1)

	switch {
	case newRef < 0:
		panic(fmt.Sprintf("Packet double free detected! ID: %d", p.ID))
	case newRef == 0:
		if p.Buffer != nil {
			if p.Buffer.Cap() <= MaxPooledBufferCap {
				bufferPool.Put(p.Buffer)
			}
			p.Buffer = nil
		}
	}
}

func getZlibReader(r io.Reader) io.ReadCloser {
	reader := zlibReaderPool.Get().(io.ReadCloser)

	// check why zlib.Reader is not exported
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

func writeFramedPacket(conn net.Conn, packetID mc.VarInt, payload []byte, threshold int) error {
	frame := bufferPool.Get().(*bytes.Buffer)
	frame.Reset()
	defer bufferPool.Put(frame)

	uncompressedSize := packetID.Len() + len(payload)
	if uncompressedSize > MaxUncompressedSize {
		return fmt.Errorf("packet size %d exceeds protocol limit", uncompressedSize)
	}

	if threshold >= 0 && uncompressedSize >= threshold {
		dataLength := mc.VarInt(uncompressedSize)
		compBuf := bufferPool.Get().(*bytes.Buffer)
		compBuf.Reset()
		defer bufferPool.Put(compBuf)

		writer := getZlibWriter(compBuf)
		_, _ = packetID.WriteTo(writer)
		_, _ = writer.Write(payload)
		_ = writer.Close()
		zlibWriterPool.Put(writer)

		packetLength := mc.VarInt(dataLength.Len() + compBuf.Len())
		_, _ = packetLength.WriteTo(frame)
		_, _ = dataLength.WriteTo(frame)
		_, _ = compBuf.WriteTo(frame)
	} else {
		packetLengthValue := uncompressedSize
		if threshold >= 0 {
			packetLengthValue += mc.VarInt(0).Len()
		}

		packetLength := mc.VarInt(packetLengthValue)
		_, _ = packetLength.WriteTo(frame)
		if threshold >= 0 {
			_, _ = mc.VarInt(0).WriteTo(frame)
		}
		_, _ = packetID.WriteTo(frame)
		_, _ = frame.Write(payload)
	}

	var err error
	if unsafeConn, ok := conn.(interface{ WriteInPlace([]byte) (int, error) }); ok {
		_, err = unsafeConn.WriteInPlace(frame.Bytes())
	} else {
		_, err = conn.Write(frame.Bytes())
	}
	return err
}
