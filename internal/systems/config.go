package systems

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
	goyaml "gopkg.in/yaml.v3"
)

type RconConfig struct {
	Enabled  bool   `yaml:"enabled" default:"false"`
	Port     int    `yaml:"port" default:"25575" min:"1" max:"65535"`
	Password string `yaml:"password" default:""`
}

type ServerConfig struct {
	Host          string `yaml:"host" default:""`
	Port          int    `yaml:"port" default:"25565" min:"1" max:"65535"`
	Motd          string `yaml:"motd" default:"A Minecraft Server"`
	LevelName     string `yaml:"level_name" default:"world"`
	MaxPlayers    int    `yaml:"max_players" default:"20" min:"1"`
	GameMode      int    `yaml:"gamemode" default:"0" min:"0" max:"3"`
	ForceGameMode bool   `yaml:"force_gamemode" default:"false"`
	Hardcore      bool   `yaml:"hardcore" default:"false"`
	RespawnScreen bool   `yaml:"respawn_screen" default:"true"`
}

type PerformanceConfig struct {
	TickRate           int `yaml:"tick_rate" default:"20" min:"1" max:"200"`
	MaxViewDistance    int `yaml:"max_view_distance" default:"10" min:"2" max:"32"`
	SimulationDistance int `yaml:"simulation_distance" default:"10" min:"2" max:"32"`
	MaxMemory          int `yaml:"max_memory" default:"0"`
}

type CompressionConfig struct {
	Enabled   bool `yaml:"enabled" default:"true"`
	Threshold int  `yaml:"threshold" default:"256" min:"0"`
}

type NetworkConfig struct {
	Compression       CompressionConfig `yaml:"compression"`
	Rcon              RconConfig        `yaml:"rcon"`
	ConnectionTimeout int               `yaml:"connection_timeout" default:"30" min:"1"`
}

type LoggingConfig struct {
	Level string `yaml:"level" default:"info"`
	File  string `yaml:"file" default:""`
}

type RateLimitConfig struct {
	MaxPacketsPerTick    int `yaml:"max_packets_per_tick" default:"64" min:"1"`
	MaxPacketSize        int `yaml:"max_packet_size" default:"2097152" min:"1"`
	ConnectionsPerSecond int `yaml:"connections_per_second" default:"10" min:"1"`
}

type WhitelistConfig struct {
	Enabled bool `yaml:"enabled" default:"false"`
	Enforce bool `yaml:"enforce" default:"false"`
}

type SecurityConfig struct {
	OnlineMode             bool            `yaml:"online_mode" default:"true"`
	SecureProfile          bool            `yaml:"secure_profile" default:"true"`
	OpLevel                int             `yaml:"op_level" default:"4" min:"1" max:"4"`
	Whitelist              WhitelistConfig `yaml:"whitelist"`
	RateLimit              RateLimitConfig `yaml:"rate_limit"`
	PreventProxyConnection bool            `yaml:"prevent_proxy_connections" default:"false"`
}

type DataFilesConfig struct {
	Whitelist     string `yaml:"whitelist" default:"whitelist.json"`
	BannedPlayers string `yaml:"banned_players" default:"banned-players.json"`
	BannedIPs     string `yaml:"banned_ips" default:"banned-ips.json"`
	Ops           string `yaml:"ops" default:"ops.json"`
	UserCache     string `yaml:"user_cache" default:"usercache.json"`
}

type PprofConfig struct {
	Enabled bool   `yaml:"enabled" default:"false"`
	Port    int    `yaml:"port" default:"6060" min:"1" max:"65535"`
	Addr    string `yaml:"addr" default:"localhost"`
}

type ProfilingConfig struct {
	Pprof PprofConfig `yaml:"pprof"`
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Network     NetworkConfig     `yaml:"network"`
	Performance PerformanceConfig `yaml:"performance"`
	Security    SecurityConfig    `yaml:"security"`
	Logging     LoggingConfig     `yaml:"logging"`
	Profiling   ProfilingConfig   `yaml:"profiling"`
	DataFiles   DataFilesConfig   `yaml:"data_files"`
}

func LoadConfig(path string, cliArgs []string) (*Config, error) {
	baseCfg, err := loadConfig(path, nil)
	if err != nil {
		return nil, err
	}

	if err := writeDefaultConfig(path, baseCfg); err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	if len(cliArgs) == 0 {
		return baseCfg, nil
	}

	return loadConfig(path, cliArgs)
}

func ReloadConfig(path string, cliArgs []string) (*Config, error) {
	return loadConfig(path, cliArgs)
}

func loadConfig(path string, cliArgs []string) (*Config, error) {
	k := koanf.New(".")

	defaults := defaultConfig()
	if err := k.Load(structs.Provider(defaults, "yaml"), nil); err != nil {
		return nil, fmt.Errorf("failed to load defaults: %w", err)
	}

	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	if overrides, err := parseCliOverrides(cliArgs); err != nil {
		return nil, fmt.Errorf("failed to apply CLI overrides: %w", err)
	} else if len(overrides) > 0 {
		if err := k.Load(confmap.Provider(overrides, "."), nil); err != nil {
			return nil, fmt.Errorf("failed to apply CLI overrides: %w", err)
		}
	}

	cfg := &Config{}
	if err := k.UnmarshalWithConf("", cfg, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	enforceConstraints(cfg)

	return cfg, nil
}

func defaultConfig() Config {
	cfg := Config{}
	applyDefaultTags(&cfg)
	return cfg
}

func applyDefaultTags(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	applyDefaultTagsToStruct(v)
}

func applyDefaultTagsToStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() == reflect.Struct {
			applyDefaultTagsToStruct(field)
			continue
		}

		def := fieldType.Tag.Get("default")
		if def == "" {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(def)
		case reflect.Int:
			if val, err := strconv.Atoi(def); err == nil {
				field.SetInt(int64(val))
			}
		case reflect.Bool:
			if val, err := strconv.ParseBool(def); err == nil {
				field.SetBool(val)
			}
		default:
		}
	}
}

func parseCliOverrides(args []string) (map[string]interface{}, error) {
	overrides := make(map[string]interface{})

	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			continue
		}

		arg = strings.TrimPrefix(arg, "--")
		eqIdx := strings.Index(arg, "=")
		if eqIdx == -1 {
			return nil, fmt.Errorf("invalid argument format %q, expected --section.key=value", arg)
		}

		key := arg[:eqIdx]
		value := arg[eqIdx+1:]

		if len(strings.Split(key, ".")) < 2 {
			return nil, fmt.Errorf("invalid argument key %q, expected section.key", key)
		}

		overrides[key] = value
	}

	return overrides, nil
}

func enforceConstraints(cfg *Config) {
	v := reflect.ValueOf(cfg).Elem()
	enforceConstraintsOnStruct(v)
}

func enforceConstraintsOnStruct(v reflect.Value) {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() == reflect.Struct {
			enforceConstraintsOnStruct(field)
			continue
		}
		if field.Kind() != reflect.Int {
			continue
		}

		val := int(field.Int())
		if minTag := fieldType.Tag.Get("min"); minTag != "" {
			if minVal, err := strconv.Atoi(minTag); err == nil && val < minVal {
				field.SetInt(int64(minVal))
			}
		}
		if maxTag := fieldType.Tag.Get("max"); maxTag != "" {
			if maxVal, err := strconv.Atoi(maxTag); err == nil && val > maxVal {
				field.SetInt(int64(maxVal))
			}
		}
	}
}

func SaveConfig(path string, cfg *Config) error {
	return writeDefaultConfig(path, cfg)
}

func writeDefaultConfig(path string, cfg *Config) error {
	data, err := goyaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
