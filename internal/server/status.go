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

	resp, _ := packet.NewPacket(packet.StatusClientboundStatusResponse, mc.String(jsonData))
	_ = resp.Send(c.Conn, c.CompressionThreshold)
	resp.Free()
}

func (c *Connection) HandlePing(timestamp *mc.Long) {
	pkt, _ := packet.NewPacket(packet.StatusClientboundPongResponse, timestamp)
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	c.close()
}
