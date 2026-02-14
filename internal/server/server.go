package server

import (
	"context"
	"fmt"
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

type Ticker = systems.Ticker
type Broadcaster = systems.Broadcaster[*Connection, *packet.Packet]
type Router = systems.DoubleRouter[mc.State, mc.VarInt, *Connection, *packet.Packet]
type Properties = systems.Properties

type Server struct {
	World       *world.World
	Ticker      *Ticker
	Broadcaster *Broadcaster
	Router      *Router
	Properties  *Properties
	Addr        string
	Connections sync.Map
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

func NewServer() *Server {
	props, err := systems.LoadProperties("server.properties")
	if err != nil {
		log.Fatalf("Failed to load server.properties: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		Properties: props,
		Addr:       fmt.Sprintf("%s:%d", props.ServerIp, props.ServerPort),
		ctx:        ctx,
		cancel:     cancel,
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
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundMovePlayerPos, (*Connection).HandleMovePlayerPos)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundMovePlayerPosRot, (*Connection).HandleMovePlayerPosRot)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundMovePlayerRot, (*Connection).HandleMovePlayerRot)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundKeepAlive, (*Connection).HandleKeepAlivePacket)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundClientTickEnd, (*Connection).HandleClientTickEnd)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundPlayerLoaded, (*Connection).HandlePlayerLoaded)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundMovePlayerStatusOnly, (*Connection).HandleMovePlayerStatusOnly)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundPlayerCommand, (*Connection).HandlePlayerCommand)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundPlayerInput, (*Connection).HandlePlayerInput)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundSwingArm, (*Connection).HandleSwingArm)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundPlayerAction, (*Connection).HandlePlayerAction)
	server.Router.RegisterHandler(mc.StatePlay, packet.PlayServerboundChat, (*Connection).HandleChat)
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
			pkt.Retain()

			select {
			case conn.OutboundPackets <- pkt:
			default:
				pkt.Free()
			}
		},
	)
	server.World = world.NewWorld()
	// todo: change how ticker work, currently it runs on 20ticks for all connections so it delays all packets
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
		timePacket.Free()
	}
}

func processIncomingPackets(s *Server) {
	currentTick := s.World.Time

	s.Connections.Range(func(key, value any) bool {
		conn := key.(*Connection)

		// todo: move to a tick start or end handler
		// todo: fix disconnect logic, loop seems to run after after the disconnect, nullptr on player
		if conn.Player != nil {
			conn.Player.Movement.PacketCount = 0
			conn.Player.Movement.LastTickX = conn.Player.Pos[0]
			conn.Player.Movement.LastTickY = conn.Player.Pos[1]
			conn.Player.Movement.LastTickZ = conn.Player.Pos[2]
		}

		for {
			select {
			case pkt := <-conn.InboundPackets:
				{
					if !conn.Player.Loaded &&
						!(pkt.ID == packet.PlayServerboundKeepAlive || pkt.ID == packet.PlayServerboundPlayerLoaded) {
						pkt.Free()
						continue
					}
					if !s.Router.Handle(conn.State, pkt.ID, conn, pkt) {
						log.Printf("Missing handler for packet %d (0x%X) in state %d\n", pkt.ID, pkt.ID, conn.State)
					}
					pkt.Free()
				}
			default:
				goto keepAlive
			}
		}

		// todo: fix, configuration keep alive cannot be reached there
	keepAlive:
		if conn.State == mc.StatePlay || conn.State == mc.StateConfiguration {
			if currentTick-conn.LastKeepAlive > KeepAliveTimeout {
				log.Printf("keep-alive timeout for %s", conn.Conn.RemoteAddr())
				conn.close()
				return true
			}

			if currentTick%KeepAliveInterval == 0 {
				conn.SendKeepAlive()
			}
		}

		return true
	})
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
