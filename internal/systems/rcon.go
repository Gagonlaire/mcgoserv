package systems

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const (
	PacketTypeResponse = 0
	PacketTypeCommand  = 2
	PacketTypeLogin    = 3
)

type RemoteConsole struct {
	addr           string
	password       string
	messageHandler func(string, func(string))
	listener       net.Listener
	mu             sync.Mutex
	connections    map[net.Conn]bool
}

func NewRemoteConsole(addr, password string, messageHandler func(string, func(string))) *RemoteConsole {
	return &RemoteConsole{
		addr:           addr,
		password:       password,
		messageHandler: messageHandler,
		connections:    make(map[net.Conn]bool),
	}
}

func (s *RemoteConsole) Start() error {
	l, err := net.Listen("tcp", s.addr)

	if err != nil {
		return err
	}
	s.listener = l
	log.Printf("RemoteConsole server listening on %s", s.addr)

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go s.handleConnection(conn)
		}
	}()

	return nil
}

func (s *RemoteConsole) Stop() {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.connections {
		_ = conn.Close()
	}
}

func (s *RemoteConsole) handleConnection(conn net.Conn) {
	s.mu.Lock()
	s.connections[conn] = true
	s.mu.Unlock()

	defer func() {
		_ = conn.Close()
		s.mu.Lock()
		delete(s.connections, conn)
		s.mu.Unlock()
	}()

	authenticated := false

	for {
		_, reqID, pType, payload, err := readPacket(conn)
		if err != nil {
			if err != io.EOF {
				log.Printf("RemoteConsole read error from %s: %v", conn.RemoteAddr(), err)
			}
			return
		}

		if !authenticated {
			if pType == PacketTypeLogin {
				if payload == s.password {
					authenticated = true

					if err := writePacket(conn, reqID, 2, ""); err != nil {
						return
					}
					log.Printf("RemoteConsole login success from %s", conn.RemoteAddr())
				} else {
					if err := writePacket(conn, -1, 2, ""); err != nil {
						return
					}
					log.Printf("RemoteConsole login failed from %s", conn.RemoteAddr())
					return
				}
			} else {
				_ = writePacket(conn, -1, 2, "")

				return
			}
		} else {
			if pType == PacketTypeCommand {
				responded := false
				s.messageHandler(payload, func(response string) {
					responded = true
					_ = writePacket(conn, reqID, PacketTypeResponse, response)
				})

				if !responded {
					if err := writePacket(conn, reqID, PacketTypeResponse, ""); err != nil {
						return
					}
				}
			}
		}
	}
}

func readPacket(conn io.Reader) (int32, int32, int32, string, error) {
	var length int32
	err := binary.Read(conn, binary.LittleEndian, &length)

	switch {
	case err != nil:
		return 0, 0, 0, "", err
	case length < 10:
		return 0, 0, 0, "", fmt.Errorf("packet too short")
	case length > 4096:
		return 0, 0, 0, "", fmt.Errorf("packet too large")
	}

	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return 0, 0, 0, "", err
	}

	buf := bytes.NewReader(data)
	var reqID int32
	var pType int32

	if err := binary.Read(buf, binary.LittleEndian, &reqID); err != nil {
		return 0, 0, 0, "", err
	}
	if err := binary.Read(buf, binary.LittleEndian, &pType); err != nil {
		return 0, 0, 0, "", err
	}

	payloadSize := length - 10
	payload := string(data[8 : 8+payloadSize])

	return length, reqID, pType, payload, nil
}

func writePacket(conn io.Writer, reqID int32, pType int32, payload string) error {
	length := int32(4 + 4 + len(payload) + 2)
	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.LittleEndian, length); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, reqID); err != nil {
		return err
	}
	if err := binary.Write(buf, binary.LittleEndian, pType); err != nil {
		return err
	}
	buf.WriteString(payload)
	buf.Write([]byte{0, 0})

	_, err := conn.Write(buf.Bytes())

	return err
}
