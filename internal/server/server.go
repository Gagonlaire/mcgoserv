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
	"github.com/Gagonlaire/mcgoserv/internal/tick"
)

const (
	ChannelSize       = 32
	KeepAliveInterval = 100 // in ticks (5 seconds at 20 TPS)
	KeepAliveTimeout  = 300 // in ticks (15 seconds at 20 TPS)
)

// BroadcastFilter is a function that determines if a connection should receive a broadcast.
// It receives the connection to check and the sender (which may be nil for server-originated broadcasts).
type BroadcastFilter func(conn *Connection, sender *Connection) bool

// Built-in broadcast filters
var (
	// FilterEveryone sends the broadcast to all connections.
	FilterEveryone BroadcastFilter = func(conn *Connection, sender *Connection) bool {
		return true
	}

	// FilterEveryoneExceptSender sends the broadcast to all connections except the sender.
	FilterEveryoneExceptSender BroadcastFilter = func(conn *Connection, sender *Connection) bool {
		return conn != sender
	}

	// FilterOnlySender sends the broadcast only to the sender.
	FilterOnlySender BroadcastFilter = func(conn *Connection, sender *Connection) bool {
		return conn == sender
	}

	// FilterPlayState sends only to connections in the Play state.
	FilterPlayState BroadcastFilter = func(conn *Connection, sender *Connection) bool {
		return conn.State == mc.StatePlay
	}
)

// FilterExcept creates a filter that excludes specific connections.
func FilterExcept(excluded ...*Connection) BroadcastFilter {
	excludeSet := make(map[*Connection]struct{}, len(excluded))
	for _, c := range excluded {
		excludeSet[c] = struct{}{}
	}
	return func(conn *Connection, sender *Connection) bool {
		_, isExcluded := excludeSet[conn]
		return !isExcluded
	}
}

// FilterOnly creates a filter that only includes specific connections.
func FilterOnly(included ...*Connection) BroadcastFilter {
	includeSet := make(map[*Connection]struct{}, len(included))
	for _, c := range included {
		includeSet[c] = struct{}{}
	}
	return func(conn *Connection, sender *Connection) bool {
		_, isIncluded := includeSet[conn]
		return isIncluded
	}
}

// FilterCombine combines multiple filters with AND logic (all must pass).
func FilterCombine(filters ...BroadcastFilter) BroadcastFilter {
	return func(conn *Connection, sender *Connection) bool {
		for _, f := range filters {
			if !f(conn, sender) {
				return false
			}
		}
		return true
	}
}

// FilterCombineOr combines multiple filters with OR logic (any must pass).
func FilterCombineOr(filters ...BroadcastFilter) BroadcastFilter {
	return func(conn *Connection, sender *Connection) bool {
		for _, f := range filters {
			if f(conn, sender) {
				return true
			}
		}
		return false
	}
}

type Server struct {
	Addr        string
	Connections sync.Map
	Ticker      *tick.Ticker
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
	LastKeepAlive   int64 // tick number of last keep-alive response
	LastKeepAliveID int64
	ctx             context.Context
	cancel          context.CancelFunc
	closeOnce       sync.Once
}

type BroadcastMessage struct {
	Packet *packet.Packet
	Sender *Connection     // nil for server-originated broadcasts
	Filter BroadcastFilter // nil defaults to FilterEveryone
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Server{
		Addr:      ":25565",
		Ticker:    tick.NewTicker(),
		Broadcast: make(chan BroadcastMessage, ChannelSize),
		Router:    NewPacketRouter(),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Register network phase handler to process any per-tick network operations
	s.Ticker.Scheduler().RegisterHandler(tick.PhaseNetwork, s.processNetworkPhase)

	return s
}

func (s *Server) NewConnection(conn net.Conn) *Connection {
	ctx, cancel := context.WithCancel(s.ctx)
	newConnection := &Connection{
		server:          s,
		Conn:            conn,
		State:           mc.StateHandshake,
		InboundPackets:  make(chan *packet.Packet, ChannelSize),
		OutboundPackets: make(chan *packet.Packet, ChannelSize),
		LastKeepAlive:   s.Ticker.CurrentTick(),
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

	// Start the tick loop in a goroutine
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
	for msg := range s.Broadcast {
		filter := msg.Filter
		if filter == nil {
			filter = FilterEveryone
		}

		s.Connections.Range(func(key, value any) bool {
			conn := key.(*Connection)
			if filter(conn, msg.Sender) {
				select {
				case conn.OutboundPackets <- msg.Packet:
				default:
					// Channel full, connection may be slow - skip to avoid blocking
					log.Printf("broadcast dropped for %s: outbound channel full", conn.Conn.RemoteAddr())
				}
			}
			return true
		})
	}
}

// processNetworkPhase is called during each tick's network phase.
// This processes all inbound packets from all connections and handles keep-alive.
func (s *Server) processNetworkPhase() {
	currentTick := s.Ticker.CurrentTick()

	s.Connections.Range(func(key, value any) bool {
		conn := key.(*Connection)

		// Process all queued inbound packets for this connection
		for {
			select {
			case pkt := <-conn.InboundPackets:
				s.Router.Handle(conn, pkt)
			default:
				// No more packets in queue
				goto keepAlive
			}
		}

	keepAlive:
		// Handle keep-alive for connections in Play or Configuration state
		if conn.State == mc.StatePlay || conn.State == mc.StateConfiguration {
			// Check for keep-alive timeout
			if currentTick-conn.LastKeepAlive > KeepAliveTimeout {
				log.Printf("keep-alive timeout for %s", conn.Conn.RemoteAddr())
				conn.close()
				return true
			}

			// Send keep-alive every KeepAliveInterval ticks
			if currentTick%KeepAliveInterval == 0 {
				keepAliveID := mc.Long(currentTick)
				var packetID int
				if conn.State == mc.StatePlay {
					packetID = 0x2B
				} else {
					packetID = 0x4
				}

				pkt, _ := packet.NewPacket(packetID, &keepAliveID)
				conn.LastKeepAliveID = int64(keepAliveID)
				select {
				case conn.OutboundPackets <- pkt:
				default:
					// Channel full
				}
			}
		}

		return true
	})
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	c := s.NewConnection(conn)

	// Only start read and write loops - packet processing is done by the ticker
	go c.ReadLoop()
	c.WriteLoop() // WriteLoop blocks until connection closes
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

// Broadcast helper methods

// BroadcastToAll sends a packet to all connected clients.
func (s *Server) BroadcastToAll(pkt *packet.Packet) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: nil,
		Filter: FilterEveryone,
	}
}

// BroadcastToAllExcept sends a packet to all connected clients except the specified ones.
func (s *Server) BroadcastToAllExcept(pkt *packet.Packet, excluded ...*Connection) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: nil,
		Filter: FilterExcept(excluded...),
	}
}

// BroadcastToOnly sends a packet only to the specified clients.
func (s *Server) BroadcastToOnly(pkt *packet.Packet, included ...*Connection) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: nil,
		Filter: FilterOnly(included...),
	}
}

// BroadcastWithFilter sends a packet using a custom filter.
func (s *Server) BroadcastWithFilter(pkt *packet.Packet, filter BroadcastFilter) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: nil,
		Filter: filter,
	}
}

// BroadcastFromSender sends a packet from a sender using a filter.
// The sender is passed to the filter function for context.
func (s *Server) BroadcastFromSender(pkt *packet.Packet, sender *Connection, filter BroadcastFilter) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: sender,
		Filter: filter,
	}
}

// BroadcastToAllPlayers sends a packet to all clients in the Play state.
func (s *Server) BroadcastToAllPlayers(pkt *packet.Packet) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: nil,
		Filter: FilterPlayState,
	}
}

// BroadcastToOtherPlayers sends a packet to all clients in Play state except the sender.
func (s *Server) BroadcastToOtherPlayers(pkt *packet.Packet, sender *Connection) {
	s.Broadcast <- BroadcastMessage{
		Packet: pkt,
		Sender: sender,
		Filter: FilterCombine(FilterPlayState, FilterEveryoneExceptSender),
	}
}
