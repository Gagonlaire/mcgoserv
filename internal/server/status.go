package server

import (
	"encoding/json"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type StatusResponsePacket struct {
	JSONResponse string `mc:"string"`
}

type PingPacket struct {
	Timestamp int64 `mc:"long"`
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

func HandleStatusPacket(conn *Connection, pkt *packet.Packet) {
	status := StatusResponse{}
	status.Version.Name = "1.21.5"
	status.Version.Protocol = 770
	status.Players.Max = 100
	status.Players.Online = 0
	status.Description.Text = "Server Go Minecraft"

	jsonData, err := json.Marshal(status)
	if err != nil {
		return
	}

	if err := pkt.Encode(0x0, &StatusResponsePacket{
		JSONResponse: string(jsonData),
	}); err != nil {
		return
	}
	_ = pkt.Send(conn.Conn)
}

func HandlePingPacket(conn *Connection, pkt *packet.Packet) {
	var ping PingPacket

	pkt.Decode(&ping)
	if err := pkt.Encode(0x1, &ping); err != nil {
		return
	}
	_ = pkt.Send(conn.Conn)
}
