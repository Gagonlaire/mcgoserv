package packet

import (
	"bytes"
	"fmt"
	"github.com/Gagonlaire/mcgoserv/internal/mcproto"
	"io"
	"net"
	"reflect"
)

type Packet struct {
	ID     int32
	Buffer *bytes.Buffer
}

func New(packetID int32) *Packet {
	return &Packet{
		ID:     packetID,
		Buffer: bytes.NewBuffer(make([]byte, 0, 256)),
	}
}

func Receive(conn net.Conn) (*Packet, error) {
	packetLength, err := mcproto.ReadVarInt(conn)
	if err != nil {
		return nil, fmt.Errorf("error reading packet length: %w", err)
	}

	packetData := make([]byte, packetLength)
	_, err = io.ReadFull(conn, packetData)
	if err != nil {
		return nil, fmt.Errorf("error reading packet data (expected %d bytes): %w", packetLength, err)
	}

	buffer := bytes.NewBuffer(packetData)
	packetID, _ := mcproto.ReadVarInt(buffer)

	return &Packet{
		ID:     packetID,
		Buffer: buffer,
	}, nil
}

func (p *Packet) Decode(s any) {
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		panic("expected a pointer to a struct")
	}
	val = val.Elem()
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := typ.Field(i).Tag.Get("mc")
		if tag == "" || !field.CanSet() {
			continue
		}
		if err := readFieldByTag(p.Buffer, field, tag); err != nil {
			panic(fmt.Sprintf("error reading field %s: %v", typ.Field(i).Name, err))
		}
	}
}

func (p *Packet) Encode(packetId int32, s any) error {
	p.Buffer.Reset()
	p.ID = packetId
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct")
	}
	val = val.Elem()
	typ := val.Type()

	if err := mcproto.WriteVarInt(p.Buffer, packetId); err != nil {
		return err
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		tag := typ.Field(i).Tag.Get("mc")
		if tag == "" {
			continue
		}
		if err := writeFieldByTag(p.Buffer, field, tag); err != nil {
			return fmt.Errorf("error writing field %s: %w", typ.Field(i).Name, err)
		}
	}
	return nil
}

func (p *Packet) Send(conn net.Conn) error {
	packetData := p.Buffer.Bytes()
	lengthBuf := &bytes.Buffer{}
	if err := mcproto.WriteVarInt(lengthBuf, int32(len(packetData))); err != nil {
		return err
	}
	if _, err := conn.Write(lengthBuf.Bytes()); err != nil {
		return err
	}
	_, err := conn.Write(packetData)
	return err
}

func readFieldByTag(r io.Reader, field reflect.Value, tag string) error {
	switch tag {
	case "varint":
		v, err := mcproto.ReadVarInt(r)
		if err != nil {
			return err
		}
		field.SetInt(int64(v))
	case "string":
		v, err := mcproto.ReadString(r)
		if err != nil {
			return err
		}
		field.SetString(v)
	case "u16":
		v, err := mcproto.ReadUInt16(r)
		if err != nil {
			return err
		}
		field.SetUint(uint64(v))
	case "long":
		v, err := mcproto.ReadLong(r)
		if err != nil {
			return err
		}
		field.SetInt(v)
	default:
		return fmt.Errorf("tag inconnu: %s", tag)
	}
	return nil
}

func writeFieldByTag(w *bytes.Buffer, field reflect.Value, tag string) error {
	switch tag {
	case "varint":
		return mcproto.WriteVarInt(w, int32(field.Int()))
	case "string":
		return mcproto.WriteString(w, field.String())
	case "u16":
		return mcproto.WriteUInt16(w, uint16(field.Uint()))
	case "long":
		return mcproto.WriteLong(w, field.Int())
	default:
		panic(fmt.Sprintf("tag inconnu: %s", tag))
	}
}
