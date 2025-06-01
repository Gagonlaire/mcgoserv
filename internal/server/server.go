package server

import (
	"github.com/Gagonlaire/mcgoserv/internal/mcproto"
	"log"
	"net"
)

type Server struct {
	Addr string
}

func New() *Server {
	return &Server{
		Addr: ":8080",
	}
}

func (s *Server) Serve() {
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
	defer conn.Close()

	handshake, err := mcproto.ReadHandshakePacket(conn)
	if err != nil {
		return
	}
	if handshake.NextState == mcproto.Status {
		err := mcproto.ReadStatusPacket(conn)

		if err != nil {
			log.Printf("error reading status packet: %v", err)
			return
		}

		err = mcproto.ReadPingRequest(conn)

		if err != nil {
			log.Printf("error reading ping request: %v", err)
			return
		}
	} else if handshake.NextState == mcproto.Login {
		// todo: implement login
	}
}
