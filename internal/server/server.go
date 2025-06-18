package server

import (
	"context"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"log"
	"net"
	"sync"
)

type State int32

const (
	StateStatus State = iota + 1
	StateLogin
	StateConfiguration
	StateTransfer
	StateHandshake
	StatePlay
)

type Server struct {
	Addr        string
	Connections map[net.Conn]*Connection
	Players     map[string]*Player
	muConn      sync.RWMutex
	muPlayers   sync.RWMutex
}

type Connection struct {
	Conn         net.Conn
	State        State
	LastPacketID mc.VarInt
	Player       *Player
}

type Player struct {
	UUID     string
	Username string
}

func NewServer() *Server {
	return &Server{
		Addr:        ":8080",
		Connections: make(map[net.Conn]*Connection),
		Players:     make(map[string]*Player),
	}
}

func (s *Server) createConnection(conn net.Conn) *Connection {
	newConnection := &Connection{
		Conn:         conn,
		State:        StateHandshake,
		LastPacketID: -1,
		Player:       nil,
	}
	s.muConn.Lock()
	s.Connections[conn] = newConnection
	s.muConn.Unlock()

	return newConnection
}

func (s *Server) closeConnection(conn *Connection) {
	if conn.Player != nil {
		// todo: additional logic like notifying other players, saving player state, etc.
		s.muPlayers.Lock()
		delete(s.Players, conn.Player.UUID)
		s.muPlayers.Unlock()
	}

	s.muConn.Lock()
	delete(s.Connections, conn.Conn)
	s.muConn.Unlock()

	// todo: check if we need to send a packet before closing
	err := conn.Conn.Close()
	if err != nil {
		log.Printf("error closing connection from %s: %v", conn.Conn.RemoteAddr(), err)
		return
	}
	log.Printf("connection from %s closed", conn.Conn.RemoteAddr())
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", s.Addr)

	if err != nil {
		log.Fatalf("failed to listen on %s: %v", s.Addr, err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("failed to accept connection: %v", err)
			continue
		}

		log.Printf("accepted connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	wrpConn := s.createConnection(conn)
	defer s.closeConnection(wrpConn)

	for {
		pkt, err := packet.Receive(wrpConn.Conn)
		if err != nil {
			// todo: check if the error is due to a closed wrpConn or a read error
			log.Printf("error reading packet from %s: %v", conn.RemoteAddr(), err)
			return
		}

		s.handlePacket(wrpConn, pkt)
		wrpConn.LastPacketID = pkt.ID
	}
}

func (s *Server) handlePacket(conn *Connection, pkt *packet.Packet) {
	switch conn.State {
	case StateHandshake:
		if pkt.ID == 0x0 {
			HandleHandshakePacket(conn, pkt)
		}
	case StateStatus:
		switch pkt.ID {
		case 0x0:
			HandleStatusPacket(conn, pkt)
		case 0x1:
			HandlePingPacket(conn, pkt)
		}
	case StateLogin:
		switch pkt.ID {
		case 0x0:
			HandleLoginStartPacket(conn, pkt)
		case 0x3:
			HandleLoginAckPacket(conn, pkt)
		}
	case StateConfiguration:
		switch pkt.ID {
		case 0x7:
			HandleClientKnownPacksPacket(conn, pkt)
		case 0x3:
			HandleFinishConfigurationAckPacket(conn, pkt)
		}
	case StateTransfer:
		context.TODO()
	}
}
