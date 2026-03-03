package player_registry

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/Gagonlaire/mcgoserv/internal/api"
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	"github.com/google/uuid"
)

type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type OpEntry struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Level               int    `json:"level"`
	BypassesPlayerLimit bool   `json:"bypassesPlayerLimit"`
}

type UserCacheEntry struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	ExpiresOn string `json:"expiresOn"`
}

type BanEntry struct {
	UUID    string `json:"uuid,omitempty"`
	Name    string `json:"name,omitempty"`
	IP      string `json:"ip,omitempty"`
	Created string `json:"created"`
	Source  string `json:"source"`
	Expires string `json:"expires"`
	Reason  string `json:"reason"`
}

type PlayerRegistry struct {
	Mu            sync.RWMutex
	Whitelist     []WhitelistEntry
	BannedPlayers []BanEntry
	BannedIPs     []BanEntry
	Ops           []OpEntry
	UserCache     []UserCacheEntry

	whitelistFile     string
	bannedPlayersFile string
	bannedIPsFile     string
	opsFile           string
	userCacheFile     string
}

func NewPlayerRegistry(whitelistFile, bannedPlayersFile, bannedIPsFile, opsFile, userCacheFile string) *PlayerRegistry {
	playerList := &PlayerRegistry{
		whitelistFile:     whitelistFile,
		bannedPlayersFile: bannedPlayersFile,
		bannedIPsFile:     bannedIPsFile,
		opsFile:           opsFile,
		userCacheFile:     userCacheFile,
	}
	playerList.Mu.Lock()
	defer playerList.Mu.Unlock()

	playerList.Whitelist = make([]WhitelistEntry, 0)
	playerList.BannedPlayers = make([]BanEntry, 0)
	playerList.BannedIPs = make([]BanEntry, 0)
	playerList.Ops = make([]OpEntry, 0)
	playerList.UserCache = make([]UserCacheEntry, 0)

	playerList.load(playerList.whitelistFile, &playerList.Whitelist)
	playerList.load(playerList.bannedPlayersFile, &playerList.BannedPlayers)
	playerList.load(playerList.bannedIPsFile, &playerList.BannedIPs)
	playerList.load(playerList.opsFile, &playerList.Ops)
	playerList.load(playerList.userCacheFile, &playerList.UserCache)

	return playerList
}

func (pl *PlayerRegistry) load(filename string, v interface{}) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := pl.save(filename, v); err != nil {
			logger.Error("Failed to create %s: %v", filename, err)
		}
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		logger.Error("Failed to read %s: %v", filename, err)
		return
	}

	if len(data) == 0 {
		return
	}
	if err := json.Unmarshal(data, v); err != nil {
		logger.Error("Failed to parse %s: %v", filename, err)
	}
}

func (pl *PlayerRegistry) save(filename string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (pl *PlayerRegistry) IsOp(UUID uuid.UUID) (bool, *OpEntry) {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	for _, entry := range pl.Ops {
		if entry.UUID == UUID.String() {
			return true, &entry
		}
	}
	return false, nil
}

func (pl *PlayerRegistry) GetOpLevel(UUID uuid.UUID) int {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	for _, entry := range pl.Ops {
		if entry.UUID == UUID.String() {
			return entry.Level
		}
	}
	return 0
}

func (pl *PlayerRegistry) AddOp(UUID uuid.UUID, name string, level int, bypassesPlayerLimit bool) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	for i, entry := range pl.Ops {
		if entry.UUID == UUID.String() {
			pl.Ops[i].Level = level
			pl.Ops[i].BypassesPlayerLimit = bypassesPlayerLimit
			_ = pl.save(pl.opsFile, pl.Ops)
			return
		}
	}

	pl.Ops = append(pl.Ops, OpEntry{
		UUID:                UUID.String(),
		Name:                name,
		Level:               level,
		BypassesPlayerLimit: bypassesPlayerLimit,
	})
	_ = pl.save(pl.opsFile, pl.Ops)
}

func (pl *PlayerRegistry) RemoveOp(name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	newOps := make([]OpEntry, 0)
	for _, entry := range pl.Ops {
		if entry.Name != name {
			newOps = append(newOps, entry)
		}
	}
	pl.Ops = newOps
	_ = pl.save(pl.opsFile, pl.Ops)
}

func (pl *PlayerRegistry) GetUserCacheProfile(name string) *UserCacheEntry {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	for _, entry := range pl.UserCache {
		if entry.Name == name {
			if entry.ExpiresOn != "" {
				expireTime, err := time.Parse("2006-01-02 15:04:05 -0700", entry.ExpiresOn)
				if err == nil && time.Now().After(expireTime) {
					return nil
				}
			}
			e := entry
			return &e
		}
	}
	return nil
}

func (pl *PlayerRegistry) AddUserCacheEntry(UUID uuid.UUID, name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	expiresOn := time.Now().AddDate(0, 1, 0).Format("2006-01-02 15:04:05 -0700")

	for i, entry := range pl.UserCache {
		if entry.UUID == UUID.String() {
			pl.UserCache[i].Name = name
			pl.UserCache[i].ExpiresOn = expiresOn
			_ = pl.save(pl.userCacheFile, pl.UserCache)
			return
		}
	}

	pl.UserCache = append(pl.UserCache, UserCacheEntry{
		UUID:      UUID.String(),
		Name:      name,
		ExpiresOn: expiresOn,
	})
	_ = pl.save(pl.userCacheFile, pl.UserCache)
}

func (pl *PlayerRegistry) GetUserUUID(name string) (uuid.UUID, error) {
	if entry := pl.GetUserCacheProfile(name); entry != nil {
		id, err := uuid.Parse(entry.UUID)
		if err == nil {
			return id, nil
		}
	}
	u, returnedName, err := api.GetUserUUID(name)
	if err != nil {
		return uuid.Nil, err
	}
	pl.AddUserCacheEntry(u, returnedName)
	return u, nil
}

func (pl *PlayerRegistry) GetUserProfile(uuid uuid.UUID, signed bool) (*api.MojangSession, error) {
	resp, err := api.GetUserProfile(uuid, signed)
	if err != nil {
		logger.Warn("Failed to fetch profile: %v", err)
		return nil, err
	}
	return resp, nil
}
