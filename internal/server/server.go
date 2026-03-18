package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"image/png"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/player-registry"
)

const (
	ChannelSize       = 64
	KeepAliveInterval = 100
	KeepAliveTimeout  = 300
)

type contextKey struct{}

// ContextKey is used to store the *Server in a context.Context.
var ContextKey = contextKey{}

type Keys struct {
	PrivateKey       *rsa.PrivateKey
	EncodedPublicKey []byte
	CertificateKeys  []*rsa.PublicKey
}

type Server struct {
	ctx               context.Context
	Ticker            *systems.Ticker
	Router            *Router
	Properties        *systems.Properties
	PlayerRegistry    *player_registry.PlayerRegistry
	RemoteConsole     *systems.RemoteConsole
	Commander         *commander.Dispatcher
	World             *world.World
	cancel            context.CancelFunc
	ConnectionsByEID  sync.Map
	Connections       sync.Map
	ID                string
	Addr              string
	Icon              string
	Keys              Keys
	wg                sync.WaitGroup
	EnforceSecureChat bool
}

func NewServer() *Server {
	s := &Server{
		PlayerRegistry: player_registry.NewPlayerRegistry(
			"whitelist.json",
			"banned-players.json",
			"banned-ips.json",
			"ops.json",
			"usercache.json",
		),
	}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ContextKey, s))
	s.ctx = ctx
	s.cancel = cancel

	s.Router = NewRouter(int(mc.StateMax))
	s.registerPacketHandlers()

	s.Commander = commander.NewDispatcher()

	s.Ticker = systems.NewTicker(mc.TicksPerSecond)
	s.registerTickerSteps()

	return s
}

func (s *Server) Start() {
	startTime := time.Now()

	logger.Info("Starting Minecraft server version %s", logger.Value(mcdata.GameVersion))

	logger.Info("Loading properties")
	props, err := systems.LoadProperties("server.properties")
	if err != nil {
		logger.Fatal("Failed to load server.properties: %v", err)
	}
	s.Properties = props
	s.Addr = fmt.Sprintf("%s:%d", props.ServerIp, props.ServerPort)
	s.EnforceSecureChat = props.EnforceSecureProfile && props.OnlineMode
	logger.Info("Default game type: %s", logger.Value(mc.GameModeString(props.GameMode)))

	logger.Info("Generating keypair")
	s.generateKeys()

	s.loadServerIcon()

	logger.Info("Starting Minecraft server on %s", logger.Network(s.Addr))
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		logger.Fatal("Failed to bind to %s: %v", logger.Network(s.Addr), err)
	}

	logger.Info("Preparing level \"%s\"", logger.Value(props.LevelName))
	s.World = world.NewWorld()
	logger.Info("Done (%s)! For help, type \"help\"", logger.Value(time.Since(startTime).Round(time.Millisecond)))

	if props.EnableRcon {
		s.startRCON()
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

func (s *Server) startRCON() {
	if s.Properties.RconPassword == "" {
		logger.Warn("No rcon password set in server.properties, rcon disabled!")
		return
	}

	s.RemoteConsole = systems.NewRemoteConsole(
		fmt.Sprintf("0.0.0.0:%d", s.Properties.RconPort),
		s.Properties.RconPassword,
		func(input string, respond func(string)) {
			src := &commander.CommandSource{
				PermissionLevel: 4,
				Server:          s,
				SendMessage: func(msg any) {
					if comp, ok := msg.(tc.Component); ok {
						respond(comp.String())
					}
				},
			}

			if _, err := s.Commander.ExecuteInput(s.ctx, src, input); err != nil {
				respond(commander.AsCommandError(err).ToComponent().String())
			}
		},
	)

	logger.Info("Starting remote control listener")
	if err := s.RemoteConsole.Start(); err != nil {
		logger.Error("Failed to start RCON server: %v", err)
	}
}

func (s *Server) Stop() {
	logger.Info("Stopping server")
	s.Ticker.Stop()
	if s.RemoteConsole != nil {
		s.RemoteConsole.Stop()
	}
	s.Connections.Range(func(k, v interface{}) bool {
		conn := k.(*Connection)
		conn.Disconnect(tc.Translatable(mcdata.MultiplayerDisconnectServerShutdown))
		return true
	})
	s.cancel()
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		logger.Warn("Shutdown timed out, forcing exit")
	}
	// todo: save world
}

func (s *Server) generateKeys() {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		logger.Fatal("Failed to generate RSA keypair: %v", err)
	}

	publicKeyDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		logger.Fatal("Error marshalling public key: %v", err)
	}

	playerCertificateKeys, err := api.GetCertificateKeys()
	if err != nil {
		logger.Warn("Failed to fetch Mojang public keys: %v", err)
	}

	s.Keys = Keys{
		PrivateKey:       key,
		EncodedPublicKey: publicKeyDER,
		CertificateKeys:  playerCertificateKeys,
	}
}

func (s *Server) loadServerIcon() {
	if _, err := os.Stat("Server-icon.png"); err == nil {
		file, err := os.Open("Server-icon.png")
		if err != nil {
			logger.Warn("Failed to open Server-icon.png: %v", err)
			return
		}
		defer file.Close()

		img, err := png.Decode(file)
		if err != nil {
			logger.Fatal("Invalid Server icon: %v", err)
			return
		}

		if img.Bounds().Dx() != 64 || img.Bounds().Dy() != 64 {
			logger.Fatal("Server icon must be 64x64 pixels")
			return
		}

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			logger.Fatal("Failed to encode Server icon: %v", err)
			return
		}

		s.Icon = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
	}
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
		timePacket, err := packet.NewPacket(packet.PlayClientboundSetTime, mc.Long(s.World.Time), mc.Long(s.World.DayTime), mc.Boolean(true))
		if err != nil {
			logger.Error("error encoding time packet: %v", err)
			return
		}

		s.World.NextTimeUpdate = s.World.Time + 20
		s.BroadcastAll(timePacket)
	}
}

func processIncomingPackets(s *Server) {
	currentTick := s.World.Time

	s.Connections.Range(func(key, value any) bool {
		c := key.(*Connection)

		// todo: move to a tick start or end handler
		if c.Player != nil {
			c.Player.Movement.PacketCount = 0
			c.Player.Movement.LastTickX = c.Player.Pos[0]
			c.Player.Movement.LastTickY = c.Player.Pos[1]
			c.Player.Movement.LastTickZ = c.Player.Pos[2]
		}

	drainPackets:
		for {
			select {
			case pkt := <-c.InboundPackets:
				pkt.Process(c, pkt.Data)
				pkt.Raw.Free()
			default:
				break drainPackets
			}
		}

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
			src := s.consoleSource()
			_, err := s.Commander.ExecuteInput(s.ctx, src, command)
			if err != nil {
				logger.Component(logger.ERROR, commander.AsCommandError(err).ToComponent())
			}
		}
	}
}

func (s *Server) consoleSource() *commander.CommandSource {
	return &commander.CommandSource{
		PermissionLevel: 4,
		Server:          s,
		SendMessage: func(msg any) {
			if comp, ok := msg.(tc.Component); ok {
				logger.Component(logger.INFO, comp)
			}
		},
	}
}
