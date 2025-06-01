package mcproto

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

func ReadRawPacket(conn net.Conn) ([]byte, error) {
	packetLength, err := ReadVarInt(conn)

	if err != nil {
		return nil, fmt.Errorf("error reading packet length: %w", err)
	}
	if packetLength <= 0 || packetLength > 2097152 {
		return nil, fmt.Errorf("invalid packet length %d", packetLength)
	}
	packetData := make([]byte, packetLength)
	_, err = io.ReadFull(conn, packetData)

	if err != nil {
		return nil, fmt.Errorf("error reading packet data (expected %d bytes): %w", packetLength, err)
	}
	return packetData, nil
}

func WritePacket(conn net.Conn, packetID int32, payloadData []byte) error {
	var packetContents bytes.Buffer
	err := WriteVarInt(&packetContents, packetID)
	if err != nil {
		return fmt.Errorf("error writing packet ID to buffer: %w", err)
	}
	if len(payloadData) > 0 {
		_, err = packetContents.Write(payloadData)
		if err != nil {
			return fmt.Errorf("error writing payload to buffer: %w", err)
		}
	}

	var finalPacketBuffer bytes.Buffer
	err = WriteVarInt(&finalPacketBuffer, int32(packetContents.Len()))
	if err != nil {
		return fmt.Errorf("error writing final packet length: %w", err)
	}
	_, err = finalPacketBuffer.Write(packetContents.Bytes())
	if err != nil {
		return fmt.Errorf("error writing final packet contents: %w", err)
	}

	_, err = conn.Write(finalPacketBuffer.Bytes())
	if err != nil {
		return fmt.Errorf("error sending packet data over connection: %w", err)
	}
	return nil
}
