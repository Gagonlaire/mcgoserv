package server

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

const (
	ChannelSize       = 32
	KeepAliveInterval = 100
	KeepAliveTimeout  = 300
)

type Server struct {
	World         *world.World
	Ticker        *systems.Ticker
	Broadcaster   *systems.Broadcaster[*Connection, *packet.Packet]
	Router        *systems.DoubleRouter[mc.State, mc.VarInt, *Connection, *packet.Packet]
	Properties    *systems.Properties
	AccessControl *systems.AccessControl
	RemoteConsole *systems.RemoteConsole
	Commander     *commander.Commander
	Addr          string
	Connections   sync.Map
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

func NewServer() *Server {
	props, err := systems.LoadProperties("server.properties")
	if err != nil {
		logger.Fatal("Failed to load server.properties: %v", err)
	}

	server := &Server{
		Properties: props,
		Addr:       fmt.Sprintf("%s:%d", props.ServerIp, props.ServerPort),
	}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "server", server))
	server.ctx = ctx
	server.cancel = cancel

	server.Router = systems.NewDoubleRouter[mc.State, mc.VarInt, *Connection, *packet.Packet]()
	server.registerPacketHandlers()

	server.Commander = commander.NewCommander()
	server.registerCommands()

	server.Ticker = systems.NewTicker(mc.TicksPerSecond)
	server.registerTickerSteps()

	server.AccessControl = systems.NewAccessControl("whitelist.json", "banned-players.json", "banned-ips.json")

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

	// todo: move world loading and logging to server start
	startTime := time.Now()
	server.World = world.NewWorld()
	duration := time.Since(startTime)
	logger.Info("Done (%.3fs)! For help, type \"help\"", duration.Seconds())

	if server.Properties.EnableRcon {
		if server.Properties.RconPassword == "" {
			logger.Warn("No rcon password set in server.properties, rcon disabled!")
		} else {
			server.RemoteConsole = systems.NewRemoteConsole(
				fmt.Sprintf("0.0.0.0:%d", server.Properties.RconPort),
				server.Properties.RconPassword,
				func(s string) string {
					resp := server.Commander.Execute(server.ctx, s)

					if resp == nil {
						return ""
					}
					return resp.String()
				},
			)
		}
	}

	return server
}

func (s *Server) Start() {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		panic(err)
	}

	go func() {
		<-s.ctx.Done()
		_ = listener.Close()
	}()
	go func() {
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
		<-stopChan
		s.Stop()
	}()
	go s.handleStdin()
	go s.Ticker.Start()

	if s.RemoteConsole != nil {
		if err := s.RemoteConsole.Start(); err != nil {
			logger.Error("Failed to start RCON server: %v", err)
		} else {
			defer s.RemoteConsole.Stop()
		}
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			if s.ctx.Err() != nil {
				return
			}
			continue
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

func (s *Server) Stop() {
	logger.Info("Stopping server")
	s.Ticker.Stop()
	s.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		conn.Disconnect(tc.Text("Server closed"))
		return true
	})
	s.cancel()
	s.wg.Wait()
	// todo: save world
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	c := s.NewConnection(conn)
	go c.ReadLoop()
	c.WriteLoop()
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
		c := key.(*Connection)

		// todo: move to a tick start or end handler
		// todo: fix disconnect logic, loop seems to run after after the disconnect, nullptr on player
		if c.Player != nil {
			c.Player.Movement.PacketCount = 0
			c.Player.Movement.LastTickX = c.Player.Pos[0]
			c.Player.Movement.LastTickY = c.Player.Pos[1]
			c.Player.Movement.LastTickZ = c.Player.Pos[2]
		}

		for {
			select {
			case pkt := <-c.InboundPackets:
				{
					if !c.Player.Loaded &&
						!(pkt.ID == packet.PlayServerboundKeepAlive || pkt.ID == packet.PlayServerboundPlayerLoaded) {
						pkt.Free()
						continue
					}
					if !s.Router.Handle(c.State, pkt.ID, c, pkt) {
						logger.Warn("Missing handler for packet %s", packet.PacketName(mc.GetStateName(c.State), "Serverbound", int(pkt.ID)))
					}
					pkt.Free()
				}
			default:
				goto keepAlive
			}
		}

		// todo: fix, configuration keep alive cannot be reached there
	keepAlive:
		if c.State == mc.StatePlay || c.State == mc.StateConfiguration {
			if currentTick-c.LastKeepAlive > KeepAliveTimeout {
				logger.Info("keep-alive timeout for %s", c.Conn.RemoteAddr())
				c.close()
				return true
			}

			if currentTick%KeepAliveInterval == 0 {
				c.SendKeepAlive()
			}
		}

		return true
	})
}

func (s *Server) handleStdin() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		command := scanner.Text()
		if strings.TrimSpace(command) != "" {
			resp := s.Commander.Execute(s.ctx, command)
			if resp != nil {
				logger.Component(logger.INFO, resp)
			}
		}
	}
}
