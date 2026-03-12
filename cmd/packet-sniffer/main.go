package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type packetFilter struct {
	Direction Direction
	ID        int
	Sample    bool
}

type Direction string

const (
	DirectionClientBound Direction = "ClientBound"
	DirectionServerBound Direction = "ServerBound"
)

var (
	/**********************************************/
	sniffedPacketFilters = []packetFilter{
		{DirectionClientBound, 0x27, true},
	}
	listenAddr = ":35565"
	targetAddr = "127.0.0.1:25565"
	/**********************************************/

	sampledPackets = make(map[string]bool)
	stdLogger      *log.Logger
	fileLogger     *log.Logger
	outputDir      = "internal/packet-sniffer"
)

func main() {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	logFilePath := filepath.Join(outputDir, "packets.txt")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to open/create packets.txt: %v", err)
	}
	defer logFile.Close()

	stdLogger = log.New(os.Stdout, "", log.LstdFlags)
	fileLogger = log.New(logFile, "", log.LstdFlags)

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", listenAddr, err)
	}
	stdLogger.Printf("Proxy listening on %s, forwarding to %s", listenAddr, targetAddr)

	for {
		clientConn, err := ln.Accept()
		if err != nil {
			stdLogger.Printf("Failed to accept connection: %v", err)
			continue
		}
		go handleConnection(clientConn, targetAddr)
	}
}

func handleConnection(clientConn net.Conn, targetAddr string) {
	defer clientConn.Close()
	serverConn, err := net.Dial("tcp", targetAddr)
	if err != nil {
		stdLogger.Printf("Failed to connect to server: %v", err)
		return
	}
	defer serverConn.Close()

	go transferMinecraftPackets(clientConn, serverConn, DirectionServerBound)
	transferMinecraftPackets(serverConn, clientConn, DirectionClientBound)
}

func transferMinecraftPackets(src, dst net.Conn, direction Direction) {
	for {
		pkt, err := packet.Receive(src, -1)
		if err != nil {
			stdLogger.Printf("Failed to receive packet %s: %v", direction, err)
			return
		}

		stdLogger.Printf("%s: ID=%d", direction, pkt.ID)

		if shouldSniff(direction, int(pkt.ID)) {
			fileLogger.Printf("Packet ID: %d, Direction: %s, Raw Bytes Length: %d;\n%v\n\n\n", pkt.ID, direction, pkt.Len(), pkt.Bytes())
		}

		if shouldSample(direction, int(pkt.ID)) {
			key := fmt.Sprintf("%02x-%s", pkt.ID, direction)
			if !sampledPackets[key] {
				filename := fmt.Sprintf("0x%02x-%s-sample.bin", pkt.ID, strings.ToLower(string(direction)))
				filePath := filepath.Join(outputDir, filename)
				if err := os.WriteFile(filePath, pkt.Bytes(), 0644); err != nil {
					panic("Failed to write sample file: " + err.Error())
				}
				sampledPackets[key] = true
			}
		}

		if err = pkt.Forward(dst, -1); err != nil {
			stdLogger.Printf("Failed to send packet %s: %v", direction, err)
			return
		}
	}
}

func shouldSniff(direction Direction, id int) bool {
	for _, f := range sniffedPacketFilters {
		if f.Direction == direction && f.ID == id {
			return true
		}
	}
	return false
}

func shouldSample(direction Direction, id int) bool {
	for _, f := range sniffedPacketFilters {
		if f.Direction == direction && f.ID == id && f.Sample {
			return true
		}
	}
	return false
}
