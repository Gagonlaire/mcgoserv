package server

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mc/world"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/packet"
	"github.com/Gagonlaire/mcgoserv/internal/systems"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/player-registry"
)

const (
	OutboundChannelSize = 16
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
	Config            *systems.Config
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
	cliArgs           []string
	wg                sync.WaitGroup
	KeepAliveTimeout  int64
	KeepAliveInterval int64
	EnforceSecureChat bool
}

func NewServer() *Server {
	s := &Server{}
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

func (s *Server) Start(args []string) {
	logger.Info("Starting Minecraft server version %s", logger.Value(mcdata.GameVersion))

	logger.Info("Loading config")
	cfg, err := systems.LoadConfig("config.yml", args)
	if err != nil {
		logger.Fatal("Failed to load config.yml: %v", err)
	}
	s.Config = cfg
	s.cliArgs = args
	if err := logger.Configure(cfg.Logging.Level, cfg.Logging.File); err != nil {
		logger.Fatal("Failed to configure logger: %v", err)
	}
	s.Addr = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	s.EnforceSecureChat = cfg.Security.SecureProfile && cfg.Security.OnlineMode
	s.KeepAliveTimeout = int64(cfg.Network.ConnectionTimeout) * mc.TicksPerSecond
	s.KeepAliveInterval = s.KeepAliveTimeout / 2
	s.PlayerRegistry = player_registry.NewPlayerRegistry(
		cfg.DataFiles.Whitelist,
		cfg.DataFiles.BannedPlayers,
		cfg.DataFiles.BannedIPs,
		cfg.DataFiles.Ops,
		cfg.DataFiles.UserCache,
	)
	logger.Info("Default game type: %s", logger.Value(mc.GameModeString(cfg.Server.GameMode)))

	if cfg.Performance.MaxMemory > 0 {
		limit := int64(cfg.Performance.MaxMemory) * 1024 * 1024
		debug.SetMemoryLimit(limit)
		logger.Info("Memory limit set to %s MB", logger.Value(cfg.Performance.MaxMemory))
	}

	if cfg.Profiling.Pprof.Enabled {
		s.startPprof()
	}

	logger.Info("Generating keypair")
	s.generateKeys()

	s.loadServerIcon()

	logger.Info("Starting Minecraft server on %s", logger.Network(s.Addr))
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		logger.Fatal("Failed to bind to %s: %v", logger.Network(s.Addr), err)
	}

	logger.Info("Preparing level \"%s\"", logger.Value(cfg.Server.LevelName))
	startTime := time.Now()
	s.World = world.NewWorld()
	logger.Info("Done (%s)! For help, type \"help\"", logger.Value(time.Since(startTime).Round(time.Millisecond)))

	if cfg.Network.Rcon.Enabled {
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
	go func() {
		reloadChan := make(chan os.Signal, 1)
		signal.Notify(reloadChan, syscall.SIGHUP)
		for {
			select {
			case <-reloadChan:
				s.reloadConfig()
			case <-s.ctx.Done():
				return
			}
		}
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

func (s *Server) startPprof() {
	pprofMux := http.NewServeMux()
	pprofMux.HandleFunc("/debug/pprof/", pprof.Index)
	pprofMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofMux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	addr := fmt.Sprintf("%s:%d", s.Config.Profiling.Pprof.Addr, s.Config.Profiling.Pprof.Port)
	pprofServer := &http.Server{
		Addr:    addr,
		Handler: pprofMux,
	}

	go func() {
		<-s.ctx.Done()
		_ = pprofServer.Close()
	}()

	logger.Info("Starting pprof server on %s", logger.Network("http://"+addr+"/debug/pprof/"))
	go func() {
		if err := pprofServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("pprof server error: %v", err)
		}
	}()
}

func (s *Server) startRCON() {
	if s.Config.Network.Rcon.Password == "" {
		logger.Warn("No rcon password set in config, rcon disabled!")
		return
	}

	s.RemoteConsole = systems.NewRemoteConsole(
		fmt.Sprintf("0.0.0.0:%d", s.Config.Network.Rcon.Port),
		s.Config.Network.Rcon.Password,
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

func (s *Server) reloadConfig() {
	logger.Info("Reloading config")
	cfg, err := systems.ReloadConfig("config.yml", s.cliArgs)
	if err != nil {
		logger.Error("Failed to reload config: %v", err)
		return
	}

	s.Config = cfg
	s.EnforceSecureChat = cfg.Security.SecureProfile && cfg.Security.OnlineMode
	s.KeepAliveTimeout = int64(cfg.Network.ConnectionTimeout) * mc.TicksPerSecond
	s.KeepAliveInterval = s.KeepAliveTimeout / 2
	logger.Info("Config reloaded successfully")
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

	logger.Debug("New connection from %s", conn.RemoteAddr())
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
			c.Player.Movement.LastTickX = c.Player.Position[0]
			c.Player.Movement.LastTickY = c.Player.Position[1]
			c.Player.Movement.LastTickZ = c.Player.Position[2]
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
			if currentTick-c.LastKeepAlive > s.KeepAliveTimeout {
				source := c.Conn.RemoteAddr().String()
				if c.Player != nil {
					source = c.Player.Name
				}
				logger.Info("%s lost connection: Timed out", logger.Identity(source))
				c.close()
				return true
			}

			if currentTick%s.KeepAliveInterval == 0 {
				c.SendKeepAlive()
			}
		}

		return true
	})
}

func flushEntityMetadata(s *Server) {
	queue := s.World.DirtyEntities
	for _, entity := range queue {
		base := entity.Base()
		if base.DimensionID == "" || !entity.HasMetaChanges() {
			entity.ClearMetaChanges()
			continue
		}

		pkt, err := packet.NewPacket(packet.PlayClientboundSetEntityData, mc.VarInt(entity.GetID()))
		if err != nil {
			entity.ClearMetaChanges()
			continue
		}
		entity.EncodeMetadata(pkt)
		_ = pkt.Encode(mc.UnsignedByte(0xFF))

		s.BroadcastEntityViewers(entity, pkt)
		entity.ClearMetaChanges()
	}
	s.World.DirtyEntities = queue[:0]
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
