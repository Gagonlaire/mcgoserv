package server

import (
	"encoding/json"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

type PlayerSample struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type StatusResponse struct {
	Description struct {
		Text string `json:"text"`
	} `json:"description"`
	Favicon string `json:"favicon,omitempty"`
	Players struct {
		Sample []PlayerSample `json:"sample,omitempty"`
		Max    int            `json:"max"`
		Online int            `json:"online"`
	} `json:"players"`
	Version struct {
		Name     string `json:"name"`
		Protocol int    `json:"protocol"`
	} `json:"version"`
	EnforceSecureChat bool `json:"enforceSecureChat"`
}

func (c *Connection) HandleStatusRequest(_ *packet.InboundPacket) {
	var data StatusResponse

	data.Version.Name = mcdata.GameVersion
	data.Version.Protocol = mcdata.ProtocolVersion
	data.Players.Max = c.Server.Properties.MaxPlayers
	data.Players.Online = c.Server.World.OnlinePlayersCount()
	data.Players.Sample = make([]PlayerSample, 0, 5)
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

	pkt := c.NewPacket(packet.StatusClientboundStatusResponse, mc.String(jsonData))
	c.SendSync(pkt)
}

func (c *Connection) HandlePing(timestamp *mc.Long) {
	pkt := c.NewPacket(packet.StatusClientboundPongResponse, timestamp)
	c.SendSync(pkt)
	c.close()
}
