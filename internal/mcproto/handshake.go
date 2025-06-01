package mcproto

import (
	"bytes"
	"encoding/json"
	"net"
)

type Intent int32

const (
	Status Intent = iota + 1
	Login
	Transfer
)

type Handshake struct {
	ProtocolVersion int32
	NextState       Intent
}

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
}

func ReadHandshakePacket(conn net.Conn) (*Handshake, error) {
	packetData, err := ReadRawPacket(conn)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(packetData)
	packetId, err := ReadVarInt(reader)
	if err != nil || packetId != 0 {
		return nil, err
	}

	protocolVersion, err := ReadVarInt(reader)
	if err != nil {
		return nil, err
	}

	if _, err := ReadString(reader); err != nil {
		return nil, err
	}
	if _, err := ReadUInt16(reader); err != nil {
		return nil, err
	}

	nextState, err := ReadVarInt(reader)
	if err != nil {
		return nil, err
	}

	return &Handshake{
		ProtocolVersion: protocolVersion,
		NextState:       Intent(nextState),
	}, nil
}

func ReadStatusPacket(conn net.Conn) error {
	packetData, err := ReadRawPacket(conn)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(packetData)
	packetId, err := ReadVarInt(reader)

	if err != nil || packetId != 0 {
		return err
	}

	status := StatusResponse{}
	status.Version.Name = "1.21.5"
	status.Version.Protocol = 770
	status.Players.Max = 100
	status.Players.Online = 0
	status.Description.Text = "Server Go Minecraft"

	jsonData, err := json.Marshal(status)
	if err != nil {
		return err
	}

	responseBuffer := bytes.NewBuffer(nil)
	if err := WriteString(responseBuffer, string(jsonData)); err != nil {
		return err
	}

	return WritePacket(conn, 0x00, responseBuffer.Bytes())
}

func ReadPingRequest(conn net.Conn) error {
	packetData, err := ReadRawPacket(conn)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(packetData)
	packetId, err := ReadVarInt(reader)
	if err != nil || packetId != 0x01 {
		return err
	}

	pingValue, err := ReadLong(reader)
	if err != nil {
		return err
	}

	pongBuffer := bytes.NewBuffer(nil)
	if err := WriteLong(pongBuffer, pingValue); err != nil {
		return err
	}

	return WritePacket(conn, 0x01, pongBuffer.Bytes())
}
