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
)

const (
	ChannelSize       = 32
	KeepAliveInterval = 100
	KeepAliveTimeout  = 300
)

type Server struct {
	Addr        string
	Connections sync.Map
	Ticker      *systems.Ticker[*Server]
	Broadcast   chan BroadcastMessage
	Router      *PacketRouter
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

type BroadcastMessage struct {
	Packet *packet.Packet
	Sender *Connection
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		Addr:      ":25565",
		Broadcast: make(chan BroadcastMessage, ChannelSize),
		Router:    NewPacketRouter(),
		ctx:       ctx,
		cancel:    cancel,
	}

	server.Ticker = systems.NewTicker(server)
	server.Ticker.RegisterHandler(processNetworkPhase)
	return server
}

func processNetworkPhase(s *Server) {
	currentTick := s.Ticker.TotalTicks

	s.Connections.Range(func(key, value any) bool {
		conn := key.(*Connection)

		for {
			select {
			case pkt := <-conn.InboundPackets:
				s.Router.Handle(conn, pkt)
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
		LastKeepAlive:   s.Ticker.TotalTicks,
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
	go s.runBroadcaster()

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

func (s *Server) runBroadcaster() {
	for _ = range s.Broadcast {
		context.TODO()
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
	close(s.Broadcast)
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
