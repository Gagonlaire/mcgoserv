package server

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
)

const (
	ChannelSize       = 32
	TickerInterval    = 50 * time.Millisecond // 20 TPS
	KeepAliveInterval = 5 * time.Second       // 5 seconds
	KeepAliveTimeout  = 15 * time.Second      // 15 seconds
)

type Server struct {
	Addr        string
	Connections sync.Map
	Ticker      *time.Ticker
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
	LastKeepAlive   time.Time
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
	return &Server{
		Addr:      ":25565",
		Ticker:    time.NewTicker(TickerInterval),
		Broadcast: make(chan BroadcastMessage, ChannelSize),
		Router:    NewPacketRouter(),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (s *Server) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)
	newConnection := &Connection{
		server:          s,
		Conn:            conn,
		State:           mc.StateHandshake,
		InboundPackets:  make(chan *packet.Packet, ChannelSize),
		OutboundPackets: make(chan *packet.Packet, ChannelSize),
		LastKeepAlive:   time.Now(),
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
	go c.WriteLoop()
	c.ProcessLoop()
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

func (c *Connection) ProcessLoop() {
	defer c.close()
	keepAliveTicker := time.NewTicker(KeepAliveInterval)
	defer keepAliveTicker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return

		case <-c.server.Ticker.C:
		processPacketBuffer:
			for {
				select {
				case pkt := <-c.InboundPackets:
					c.server.Router.Handle(c, pkt)
				default:
					break processPacketBuffer
				}
			}

		case <-keepAliveTicker.C:
			if time.Since(c.LastKeepAlive) > KeepAliveTimeout {
				log.Printf("keep-alive timeout for %s", c.Conn.RemoteAddr())
				return
			}
			keepAliveID := mc.Long(time.Now().Unix())
			if c.State == mc.StatePlay || c.State == mc.StateConfiguration {
				var packetID int
				if c.State == mc.StatePlay {
					packetID = 0x2B
				} else {
					packetID = 0x4
				}

				pkt, _ := packet.NewPacket(packetID, &keepAliveID)
				c.LastKeepAliveID = int64(keepAliveID)
				c.OutboundPackets <- pkt
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
