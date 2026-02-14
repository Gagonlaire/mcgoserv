package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/world"
	"github.com/Tnze/go-mc/nbt"
)

type Connection struct {
	server          *Server
	Player          *world.Player
	Conn            net.Conn
	State           mc.State
	InboundPackets  chan *packet.Packet
	OutboundPackets chan *packet.Packet
	LastKeepAlive   int64
	LastKeepAliveID int64
	ctx             context.Context
	cancel          context.CancelFunc
	closeOnce       sync.Once
}

func (s *Server) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)
	newConnection := &Connection{
		server:          s,
		Conn:            conn,
		State:           mc.StateHandshake,
		InboundPackets:  make(chan *packet.Packet, ChannelSize),
		OutboundPackets: make(chan *packet.Packet, ChannelSize),
		LastKeepAlive:   s.World.Time,
		ctx:             ctx,
		cancel:          cancel,
	}

	s.Connections.Store(newConnection, struct{}{})
	return newConnection
}

func (c *Connection) ReadLoop() {
	defer c.close()
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		pkt, err := packet.Receive(c.Conn)
		if err != nil {
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				log.Printf("error reading packet from %s: %v", c.Conn.RemoteAddr(), err)
			}
			return
		}
		// If not in play state, we don't use the ticking system to avoid artificial delay of packets
		if c.State == mc.StatePlay {
			c.InboundPackets <- pkt
		} else {
			if !c.server.Router.Handle(c.State, pkt.ID, c, pkt) {
				log.Printf("Missing handler for packet %s\n", packet.PacketName(mc.GetStateName(c.State), "Serverbound", int(pkt.ID)))
			}
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
			err := pkt.Send(c.Conn)
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

func (c *Connection) Send(pkt *packet.Packet) {
	select {
	case c.OutboundPackets <- pkt:
	default:
		pkt.Free()
	}
}

func (c *Connection) Disconnect(reason string) {
	// todo: create JSON text component and text component
	reasonObj := struct {
		Text string `json:"text" nbt:"text"`
	}{Text: reason}
	reasonBytes, _ := json.Marshal(reasonObj)
	var pkt *packet.Packet

	switch c.State {
	case mc.StateLogin:
		pkt, _ = packet.NewPacket(packet.LoginClientboundLoginDisconnect, mc.String(reasonBytes))
	case mc.StateConfiguration:
		pkt, _ = packet.NewPacket(packet.ConfigurationClientboundDisconnect, mc.String(reasonBytes))
	case mc.StatePlay:
		pkt, _ = packet.NewPacket(packet.PlayClientboundDisconnect)

		encoder := nbt.NewEncoder(pkt.Buffer)
		encoder.NetworkFormat(true)
		_ = encoder.Encode(reasonObj, "")
	default:
		panic("unhandled default case")
	}

	c.Send(pkt)
}

func (c *Connection) close() {
	// todo: same here, move to a flexible type
	type PlayerLeft struct {
		Text  string `nbt:"text"`
		Color string `nbt:"color,omitempty"`
	}

	c.closeOnce.Do(func() {
		c.cancel()
		if c.Player != nil {
			component := PlayerLeft{
				Text:  string(c.Player.Name) + " left the game",
				Color: "yellow",
			}

			eID := mc.VarInt(c.Player.EntityID)
			UUID := mc.UUID(c.Player.UUID)
			pkt1, _ := packet.NewPacket(packet.PlayClientboundPlayerInfoRemove, mc.VarInt(1), &UUID)
			pkt2, _ := packet.NewPacket(packet.PlayClientboundRemoveEntities, mc.VarInt(1), eID)
			pkt3, _ := packet.NewPacket(packet.PlayClientboundSystemChat)
			encoder := nbt.NewEncoder(pkt3.Buffer)
			encoder.NetworkFormat(true)
			_ = encoder.Encode(component, "")
			_ = pkt3.Encode(mc.Boolean(false))
			c.server.Broadcaster.Broadcast(pkt1, systems.NotSender(c))
			c.server.Broadcaster.Broadcast(pkt2, systems.NotSender(c))
			c.server.Broadcaster.Broadcast(pkt3, systems.NotSender(c))
			c.server.World.RemovePlayer(c.Player.UUID)
		}
		c.server.Connections.Delete(c)
		_ = c.Conn.Close()
	})
}
