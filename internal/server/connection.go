package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/container"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type QueuedPacket struct {
	Process func(*Connection, any)
	Data    any
	Raw     *packet.InboundPacket
}

type Connection struct {
	Conn            net.Conn
	ctx             context.Context
	ContextData     map[string]interface{}
	Player          *entity.Player
	Server          *Server
	InboundPackets  chan QueuedPacket
	OutboundPackets chan *packet.OutboundPacket
	cancel          context.CancelFunc
	// Cursor is the drag-in-flight item stack, transient per-window UI state.
	// TODO: wire ClickContainer handler to mutate Cursor on click/drag.
	// TODO: on CloseContainer (client or server initiated), drop Cursor into the world as an item entity, then zero it.
	// TODO: on disconnect (close()), drop Cursor into the world as an item entity before connection teardown.
	// TODO: on window open, defensively zero Cursor to prevent leakage between containers.
	Cursor               container.Slot
	State                mc.State
	CompressionThreshold int
	LastKeepAliveID      int64
	LastKeepAliveSent    time.Time
	KeepAlivePending     bool
	closeOnce            sync.Once
	writeMu              sync.Mutex
}

func (c *Connection) SetCompressionThreshold(threshold int) {
	c.writeMu.Lock()
	c.CompressionThreshold = threshold
	c.writeMu.Unlock()
}

func (s *Server) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)
	newConnection := &Connection{
		Server:               s,
		Conn:                 conn,
		State:                mc.StateHandshake,
		InboundPackets:       make(chan QueuedPacket, s.Config.Security.RateLimit.MaxPacketsPerTick),
		OutboundPackets:      make(chan *packet.OutboundPacket, OutboundChannelSize),
		CompressionThreshold: -1,
		LastKeepAliveSent:    time.Now(),
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
			if c.ctx.Err() != nil {
				return
			}
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				logger.Error("error reading packet from %s: %v", c.Conn.RemoteAddr(), err)
			}
			return
		}

		if logger.IsDebug() && !(c.State == mc.StatePlay && int(pkt.ID) == packet.PlayServerboundClientTickEnd) {
			source := c.Conn.RemoteAddr().String()
			if c.Player != nil {
				source = c.Player.Name
			}
			logger.Debug("%s -> Server: %s(0x%x)",
				source, packet.PacketName(mc.GetStateName(c.State), "Serverbound", int(pkt.ID)), pkt.ID)
		}

		if handler, ok := c.Server.Router.Get(c.State, int(pkt.ID)); ok {
			var data any

			if handler.Decode != nil {
				data, err = handler.Decode(pkt)
				if err != nil || pkt.Remaining() > 0 {
					c.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectInvalidPacket))
					return
				}
			} else {
				data = pkt
			}

			if handler.Ticked {
				select {
				case c.InboundPackets <- QueuedPacket{
					Process: handler.Process,
					Data:    data,
					Raw:     pkt,
				}:
				case <-c.ctx.Done():
					pkt.Free()
					return
				default:
					// max packets per tick exceeded, dropping packets
				}
			} else {
				handler.Process(c, data)
				pkt.Free()
			}
		} else {
			logger.Warn("Missing handler for packet: %s%s",
				logger.FmtWarn(packet.PacketName(mc.GetStateName(c.State), "Serverbound", int(pkt.ID))),
				logger.FmtWarn(fmt.Sprintf("(0x%x)", pkt.ID)),
			)
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
			if logger.IsDebug() {
				target := c.Conn.RemoteAddr().String()
				if c.Player != nil {
					target = c.Player.Name
				}
				logger.Debug("Server -> %s: %s(0x%x)",
					target, packet.PacketName(mc.GetStateName(c.State), "Clientbound", int(pkt.ID)), pkt.ID)
			}
			err := c.SendSync(pkt)
			if err != nil {
				if err != io.EOF && !errors.Is(err, net.ErrClosed) {
					logger.Error("error sending packet to %s: %v", c.Conn.RemoteAddr(), err)
				}
				return
			}
		default:
			// connection is too slow, dropping packets
		}
	}
}

// Send sends a packet asynchronously. Takes ownership of the packet.
func (c *Connection) Send(pkt *packet.OutboundPacket) {
	if pkt == nil {
		return
	}
	select {
	case c.OutboundPackets <- pkt:
		return
	case <-c.ctx.Done():
		return
	}
}

// SendSync sends a packet synchronously, blocking until it's sent. Takes ownership of the packet.
func (c *Connection) SendSync(pkt *packet.OutboundPacket) error {
	if pkt == nil {
		return nil
	}
	c.writeMu.Lock()
	err := pkt.Send(c.Conn, c.CompressionThreshold)
	c.writeMu.Unlock()
	pkt.Free()
	return err
}

// NewPacket creates a new outbound packet, logging and returning nil on encoding error.
func (c *Connection) NewPacket(ID int, fields ...io.WriterTo) *packet.OutboundPacket {
	pkt, err := packet.NewPacket(ID, fields...)
	if err != nil {
		logger.Error("error encoding packet %s: %v",
			packet.PacketName(mc.GetStateName(c.State), "Clientbound", ID), err)
		return nil
	}
	return pkt
}

func (c *Connection) Disconnect(reason tc.Component) {
	var pkt *packet.OutboundPacket

	switch c.State {
	case mc.StateLogin:
		pkt = c.NewPacket(packet.LoginClientboundLoginDisconnect, proto.String(reason.ToJSON()))
	case mc.StateConfiguration:
		pkt = c.NewPacket(packet.ConfigurationClientboundDisconnect, proto.String(reason.ToJSON()))
	case mc.StatePlay:
		pkt = c.NewPacket(packet.PlayClientboundDisconnect, reason)
	default:
		c.close()
		return
	}

	_ = c.SendSync(pkt)
	c.close()
}

func (c *Connection) close() {
	c.closeOnce.Do(func() {
		c.cancel()
		if logger.IsDebug() {
			logger.Debug("Closing connection %s (state=%s)", c.Conn.RemoteAddr(), mc.GetStateName(c.State))
		}
		if c.Player != nil {
			logger.Info("%s lost connection: Disconnected", logger.Identity(c.Player.Name))
			c.Server.ConnectionsByEID.Delete(c.Player.EntityID)
			infoRemove := c.NewPacket(packet.PlayClientboundPlayerInfoRemove, proto.VarInt(1), proto.UUID(c.Player.UUID))
			leftMessage := tc.Translatable(
				mcdata.MultiplayerPlayerLeft,
				tc.PlayerName(c.Player.Name),
			).SetColor(tc.ColorYellow)
			systemChat := c.NewPacket(packet.PlayClientboundSystemChat, leftMessage, proto.Boolean(false))

			c.Server.BroadcastOthers(c, infoRemove)
			c.Server.BroadcastOthers(c, systemChat)
			logger.Component(logger.INFO, leftMessage)
			c.Server.DespawnPlayer(c.Player)
		}
		c.Server.Connections.Delete(c)
		_ = c.Conn.Close()
	})
}
