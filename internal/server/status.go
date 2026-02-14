package server

import (
	"encoding/json"
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int `json:"max"`
		Online int `json:"online"`
		Sample []struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		} `json:"sample,omitempty"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	EnforceSecureChat bool `json:"enforceSecureChat"`
}

func (c *Connection) HandleStatusRequestPacket(pkt *packet.Packet) {
	var data StatusResponse

	data.Version.Name = mc.GameVersion
	data.Version.Protocol = mc.ProtocolVersion
	data.Players.Max = c.server.Properties.MaxPlayers
	data.Players.Online = len(c.server.World.Players)
	for _, p := range c.server.World.Players {
		if len(data.Players.Sample) >= 5 {
			break
		}
		data.Players.Sample = append(data.Players.Sample, struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		}{
			Name: string(p.Name),
			ID:   p.UUID.String(),
		})
	}
	data.Description.Text = c.server.Properties.Motd
	data.EnforceSecureChat = false
	jsonData, _ := json.Marshal(data)

	_ = pkt.ResetWith(packet.StatusClientboundStatusResponse, mc.String(jsonData))
	_ = pkt.Send(c.Conn)
}

func (c *Connection) HandlePingPacket(pkt *packet.Packet) {
	var timestamp mc.Long

	if err := pkt.Decode(&timestamp); err != nil {
		fmt.Println("Error decoding ping packet:", err)
		return
	}

	_ = pkt.ResetWith(packet.StatusClientboundPongResponse, &timestamp)
	_ = pkt.Send(c.Conn)
	c.close()
}
