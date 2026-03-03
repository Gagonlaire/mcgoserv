package server

import (
	"encoding/json"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type PlayerSample struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type StatusResponse struct {
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	Players struct {
		Max    int            `json:"max"`
		Online int            `json:"online"`
		Sample []PlayerSample `json:"sample,omitempty"`
	} `json:"players"`
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Favicon           string `json:"favicon,omitempty"`
	EnforceSecureChat bool   `json:"enforceSecureChat"`
}

func (c *Connection) HandleStatusRequest(pkt *packet.Packet) {
	var data StatusResponse

	data.Version.Name = mcdata.GameVersion
	data.Version.Protocol = mcdata.ProtocolVersion
	data.Players.Max = c.Server.Properties.MaxPlayers
	data.Players.Online = c.Server.World.OnlinePlayersCount()
	for _, player := range c.Server.World.PlayersByUUID {
		if len(data.Players.Sample) >= 5 {
			break
		}
		if player.Information.AllowServerListings {
			data.Players.Sample = append(data.Players.Sample, PlayerSample{
				Name: player.Name,
				ID:   player.UUID.String(),
			})
		}
	}
	data.Description.Text = c.Server.Properties.Motd
	if c.Server.Icon != "" {
		data.Favicon = c.Server.Icon
	}
	data.EnforceSecureChat = false
	jsonData, _ := json.Marshal(data)

	_ = pkt.ResetWith(packet.StatusClientboundStatusResponse, mc.String(jsonData))
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}

func (c *Connection) HandlePing(pkt *packet.Packet) {
	var timestamp mc.Long

	if err := pkt.Decode(&timestamp); err != nil {
		logger.Error("Error decoding ping packet: %v", err)
		return
	}

	_ = pkt.ResetWith(packet.StatusClientboundPongResponse, &timestamp)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	c.close()
}
