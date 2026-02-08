package server

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/world"
)

const (
	ChannelSize       = 32
	KeepAliveInterval = 100
	KeepAliveTimeout  = 300
)

type Broadcaster = systems.Broadcaster[*Connection, *packet.Packet]
type Router = systems.DoubleRouter[mc.State, mc.VarInt, *Connection, *packet.Packet]

type Server struct {
	World       *world.World
	Ticker      *systems.Ticker
	Broadcaster *Broadcaster
	Router      *Router
	Addr        string
	Connections sync.Map
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

type Connection struct {
	server          *Server
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

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		Addr:   ":25565",
		ctx:    ctx,
		cancel: cancel,
	}

	server.Router = systems.NewDoubleRouter[mc.State, mc.VarInt, *Connection, *packet.Packet]()
	server.Router.RegisterHandler(mc.StateHandshake, packet.HandshakeServerboundHandshake, (*Connection).HandleHandshakePacket)
	server.Router.RegisterHandler(mc.StateStatus, packet.StatusServerboundStatusRequest, (*Connection).HandleStatusRequestPacket)
	server.Router.RegisterHandler(mc.StateStatus, packet.StatusServerboundPing, (*Connection).HandlePingPacket)
	server.Router.RegisterHandler(mc.StateLogin, packet.LoginServerboundLoginStart, (*Connection).HandleLoginStartPacket)
	server.Router.RegisterHandler(mc.StateLogin, packet.LoginServerboundLoginAcknowledged, (*Connection).HandleLoginAckPacket)
	server.Router.RegisterHandler(mc.StateConfiguration, packet.ConfigurationServerboundAcknowledgeFinishConfiguration, (*Connection).HandleFinishConfigurationAckPacket)
	server.Router.RegisterHandler(mc.StateConfiguration, packet.ConfigurationServerboundKeepAlive, (*Connection).HandleKeepAlivePacket)
	server.Router.RegisterHandler(mc.StateConfiguration, packet.ConfigurationServerboundKnownPacks, (*Connection).HandleClientKnownPacksPacket)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundConfirmTeleportation, (*Connection).HandleConfirmTeleportationPacket)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundKeepAlive, (*Connection).HandleKeepAlivePacket)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundClientTickEnd, (*Connection).HandleClientTickEnd)
	// todo: maybe add debug logs after registering handlers

	server.Broadcaster = systems.NewBroadcaster(
		func(yield func(*Connection) bool) {
			server.Connections.Range(func(key, value any) bool {
				conn := key.(*Connection)
				if conn.State == mc.StatePlay {
					return yield(conn)
				}
				return true
			})
		},
		func(conn *Connection, pkt *packet.Packet) {
			select {
			case conn.OutboundPackets <- pkt:
			default:
				// todo: handle full channel
			}
		},
	)
	server.World = world.NewWorld()
	server.Ticker = systems.NewTicker(mc.TicksPerSecond)
	server.Ticker.RegisterHandler(func() { updateTime(server) })
	server.Ticker.RegisterHandler(func() { processIncomingPackets(server) })

	return server
}

func updateTime(s *Server) {
	s.World.Time++
	s.World.DayTime = (s.World.DayTime + 1) % mc.TicksPerDay

	if s.World.DayTime == 0 {
		s.World.Day++
	}

	if s.World.Time >= s.World.NextTimeUpdate {
		worldAge := mc.Long(s.World.Time)
		timeOfDay := mc.Long(s.World.DayTime)
		timeOfDayIncreasing := mc.Boolean(true)
		timePacket, _ := packet.NewPacket(packet.PlayClientboundSetTime, &worldAge, &timeOfDay, &timeOfDayIncreasing)

		s.World.NextTimeUpdate = s.World.Time + 20
		s.Broadcaster.Broadcast(timePacket)
	}
}

func processIncomingPackets(s *Server) {
	currentTick := s.World.Time

	s.Connections.Range(func(key, value any) bool {
		conn := key.(*Connection)

		for {
			select {
			case pkt := <-conn.InboundPackets:
				s.Router.Handle(conn.State, pkt.ID, conn, pkt)
			default:
				goto keepAlive
			}
		}

	keepAlive:
		if conn.State == mc.StatePlay || conn.State == mc.StateConfiguration {
			if currentTick-conn.LastKeepAlive > KeepAliveTimeout {
				log.Printf("keep-alive timeout for %s", conn.Conn.RemoteAddr())
				conn.close()
				return true
			}

			if currentTick%KeepAliveInterval == 0 {
				keepAliveID := mc.Long(currentTick)
				var packetID int
				// todo: replace with packet constants
				if conn.State == mc.StatePlay {
					packetID = 0x2B
				} else {
					packetID = 0x4
				}

				pkt, _ := packet.NewPacket(packetID, &keepAliveID)
				conn.LastKeepAliveID = int64(keepAliveID)
				select {
				case conn.OutboundPackets <- pkt:
				}
			}
		}

		return true
	})
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

func (s *Server) Start() {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", s.Addr, err)
	}
	defer listener.Close()

	go s.Ticker.Start()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			log.Printf("failed to accept connection: %v", err)
			continue
		}
		log.Printf("accepted connection from %s", conn.RemoteAddr())

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	c := s.NewConnection(conn)

	go c.ReadLoop()
	c.WriteLoop()
}

func (s *Server) Stop() {
	s.cancel()
	s.wg.Wait()
	s.Ticker.Stop()
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
		c.InboundPackets <- pkt
	}
}

func (c *Connection) WriteLoop() {
	defer c.close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case pkt := <-c.OutboundPackets:
			if err := pkt.Send(c.Conn); err != nil {
				if err != io.EOF && !errors.Is(err, net.ErrClosed) {
					log.Printf("error sending packet from %s: %v", c.Conn.RemoteAddr(), err)
				}
				return
			}
		}
	}
}

func (c *Connection) close() {
	c.closeOnce.Do(func() {
		c.cancel()
		c.server.Connections.Delete(c)
		_ = c.Conn.Close()
	})
}
