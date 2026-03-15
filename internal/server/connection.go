package server

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
)

type QueuedPacket struct {
	Process func(*Connection, any)
	Data    any
	Raw     *packet.InboundPacket
}

type Connection struct {
	Conn                 net.Conn
	ctx                  context.Context
	ContextData          map[string]interface{}
	Player               *entities.Player
	Server               *Server
	InboundPackets       chan QueuedPacket
	OutboundPackets      chan *packet.OutboundPacket
	cancel               context.CancelFunc
	State                mc.State
	CompressionThreshold int
	LastKeepAliveID      int64
	LastKeepAlive        int64
	closeOnce            sync.Once
}

func (s *Server) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)
	newConnection := &Connection{
		Server:               s,
		Conn:                 conn,
		State:                mc.StateHandshake,
		InboundPackets:       make(chan QueuedPacket, ChannelSize),
		OutboundPackets:      make(chan *packet.OutboundPacket, ChannelSize),
		CompressionThreshold: -1,
		LastKeepAlive:        s.World.Time,
		ctx:                  ctx,
		cancel:               cancel,
		ContextData:          make(map[string]interface{}),
	}

	s.Connections.Store(newConnection, struct{}{})
	return newConnection
}

func (c *Connection) ReadLoop() {
	defer c.close()
	for {
		pkt, err := packet.Receive(c.Conn, c.CompressionThreshold)
		if err != nil {
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				logger.Error("error reading packet from %s: %v", c.Conn.RemoteAddr(), err)
			}
			return
		}

		if handler, ok := c.Server.Router.Get(c.State, int(pkt.ID)); ok {
			var data any

			if handler.Decode != nil {
				data, err = handler.Decode(pkt)
				if err != nil {
					// todo: disconnect with clean reason
					c.close()
					continue
				}
				// todo: check if data remains, if so, disconnect with clean reason
			} else {
				data = pkt
			}

			if handler.Ticked {
				c.InboundPackets <- QueuedPacket{
					Process: handler.Process,
					Data:    data,
					Raw:     pkt,
				}
			} else {
				handler.Process(c, data)
				pkt.Free()
			}
		} else {
			logger.Warn("Missing handler for packet %s", packet.PacketName(mc.GetStateName(c.State), "Serverbound", int(pkt.ID)))
			pkt.Free()
		}
	}
}

func (c *Connection) WriteLoop() {
	defer c.close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case pkt := <-c.OutboundPackets:
			err := pkt.Send(c.Conn, c.CompressionThreshold)
			id := pkt.ID
			pkt.Free()
			if err != nil {
				if err != io.EOF && !errors.Is(err, net.ErrClosed) {
					log.Printf("error sending packet from %s: %v", c.Conn.RemoteAddr(), err)
				}
				return
			}

			if (c.State == mc.StateLogin && id == packet.LoginClientboundLoginDisconnect) ||
				(c.State == mc.StateConfiguration && id == packet.ConfigurationClientboundDisconnect) ||
				(c.State == mc.StatePlay && id == packet.PlayClientboundDisconnect) {
				return
			}
		}
	}
}

func (c *Connection) Send(pkt *packet.OutboundPacket) {
	select {
	case c.OutboundPackets <- pkt:
		return
	case <-c.ctx.Done():
		return
	}
}

func (c *Connection) SendSync(pkt *packet.OutboundPacket) {
	_ = pkt.Send(c.Conn, c.CompressionThreshold)
	pkt.Free()
}

func (c *Connection) Disconnect(reason tc.Component) {
	var pkt *packet.OutboundPacket

	switch c.State {
	case mc.StateLogin:
		pkt, _ = packet.NewPacket(packet.LoginClientboundLoginDisconnect, mc.String(reason.ToJSON()))
	case mc.StateConfiguration:
		pkt, _ = packet.NewPacket(packet.ConfigurationClientboundDisconnect, mc.String(reason.ToJSON()))
	case mc.StatePlay:
		pkt, _ = packet.NewPacket(packet.PlayClientboundDisconnect, reason)
	default:
		return
	}

	_ = pkt.Send(c.Conn, c.CompressionThreshold)
}

func (c *Connection) close() {
	c.closeOnce.Do(func() {
		c.cancel()
		if c.Player != nil {
			logger.Info("%s lost connection: Disconnected", logger.Identity(c.Player.Name))
			delete(c.Server.ConnectionsByEID, c.Player.EntityID)
			infoRemove, _ := packet.NewPacket(packet.PlayClientboundPlayerInfoRemove, mc.VarInt(1), mc.UUID(c.Player.UUID))
			entityRemove, _ := packet.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), mc.VarInt(c.Player.EntityID))
			leftMessage := tc.Translatable(
				mcdata.MultiplayerPlayerLeft,
				tc.PlayerName(c.Player.Name),
			).SetColor(tc.ColorYellow)
			systemChat, _ := packet.NewPacket(packet.PlayClientboundSystemChat, leftMessage, mc.Boolean(false))

			c.Server.Broadcaster.Broadcast(infoRemove, systems.NotSender(c))
			c.Server.Broadcaster.Broadcast(entityRemove, systems.NotSender(c))
			c.Server.Broadcaster.Broadcast(systemChat, systems.NotSender(c))
			logger.Component(logger.INFO, leftMessage)
			infoRemove.Free()
			entityRemove.Free()
			systemChat.Free()
			c.Server.World.RemoveEntityByUUID(c.Player.UUID)
		}
		c.Server.Connections.Delete(c)
		_ = c.Conn.Close()
	})
}
