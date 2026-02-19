package systems

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

type WhitelistEntry struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
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

type AccessControl struct {
	Mu            sync.RWMutex
	Whitelist     []WhitelistEntry
	BannedPlayers []BanEntry
	BannedIPs     []BanEntry
	// todo: add cache files for uuids

	whitelistFile     string
	bannedPlayersFile string
	bannedIPsFile     string
}

func NewAccessControl(whitelistFile, bannedPlayersFile, bannedIPsFile string) *AccessControl {
	playerList := &AccessControl{
		whitelistFile:     whitelistFile,
		bannedPlayersFile: bannedPlayersFile,
		bannedIPsFile:     bannedIPsFile,
	}
	playerList.Mu.Lock()
	defer playerList.Mu.Unlock()

	playerList.Whitelist = make([]WhitelistEntry, 0)
	playerList.BannedPlayers = make([]BanEntry, 0)
	playerList.BannedIPs = make([]BanEntry, 0)
	playerList.load(playerList.whitelistFile, &playerList.Whitelist)
	playerList.load(playerList.bannedPlayersFile, &playerList.BannedPlayers)
	playerList.load(playerList.bannedIPsFile, &playerList.BannedIPs)

	return playerList
}

func (pl *AccessControl) load(filename string, v interface{}) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := pl.save(filename, v); err != nil {
			fmt.Printf("Failed to create %s: %v\n", filename, err)
		}
		return
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", filename, err)
		return
	}

	if len(data) == 0 {
		return
	}
	if err := json.Unmarshal(data, v); err != nil {
		fmt.Printf("Failed to parse %s: %v\n", filename, err)
	}
}

func (pl *AccessControl) save(filename string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (pl *AccessControl) IsWhitelisted(UUID uuid.UUID) bool {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	for _, entry := range pl.Whitelist {
		if entry.UUID == UUID.String() {
			return true
		}
	}
	return false
}

func (pl *AccessControl) IsBanned(UUID uuid.UUID) (bool, *BanEntry) {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	now := time.Now()
	for _, entry := range pl.BannedPlayers {
		if entry.UUID == UUID.String() {
			if entry.Expires != "forever" && entry.Expires != "" {
				expireTime, err := time.Parse("2006-01-02 15:04:05 -0700", entry.Expires)
				if err == nil && now.After(expireTime) {
					// todo: check if entry shold be removed from the list
					continue
				}
			}
			return true, &entry
		}
	}
	return false, nil
}

func (pl *AccessControl) IsIPBanned(ip string) (bool, *BanEntry) {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	host, _, err := net.SplitHostPort(ip)
	if err == nil {
		ip = host
	}

	now := time.Now()
	for _, entry := range pl.BannedIPs {
		if entry.IP == ip {
			if entry.Expires != "forever" && entry.Expires != "" {
				expireTime, err := time.Parse("2006-01-02 15:04:05 -0700", entry.Expires)
				if err == nil && now.After(expireTime) {
					continue
				}
			}
			return true, &entry
		}
	}
	return false, nil
}

func (pl *AccessControl) AddWhitelist(UUID uuid.UUID, name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	for _, entry := range pl.Whitelist {
		if entry.UUID == UUID.String() {
			return
		}
	}

	pl.Whitelist = append(pl.Whitelist, WhitelistEntry{UUID: UUID.String(), Name: name})
	_ = pl.save(pl.whitelistFile, pl.Whitelist)
}

func (pl *AccessControl) RemoveWhitelist(name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	newWhitelist := make([]WhitelistEntry, 0)
	for _, entry := range pl.Whitelist {
		if entry.Name != name {
			newWhitelist = append(newWhitelist, entry)
		}
	}
	pl.Whitelist = newWhitelist
	_ = pl.save(pl.whitelistFile, pl.Whitelist)
}

func (pl *AccessControl) Ban(UUID uuid.UUID, name, source, reason, expires string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	entry := BanEntry{
		UUID:    UUID.String(),
		Name:    name,
		Created: time.Now().Format("2006-01-02 15:04:05 -0700"),
		Source:  source, // source is Rcon, Server or the player name
		Expires: expires,
		Reason:  reason,
	}

	pl.BannedPlayers = append(pl.BannedPlayers, entry)
	_ = pl.save(pl.bannedPlayersFile, pl.BannedPlayers)
}

func (pl *AccessControl) BanIP(ip, source, reason, expires string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	entry := BanEntry{
		IP:      ip,
		Created: time.Now().Format("2006-01-02 15:04:05 -0700"),
		Source:  source,
		Expires: expires,
		Reason:  reason,
	}

	pl.BannedIPs = append(pl.BannedIPs, entry)
	_ = pl.save(pl.bannedIPsFile, pl.BannedIPs)
}

func (pl *AccessControl) Unban(name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	newBannedPlayers := make([]BanEntry, 0)
	for _, entry := range pl.BannedPlayers {
		if entry.Name != name {
			newBannedPlayers = append(newBannedPlayers, entry)
		}
	}
	pl.BannedPlayers = newBannedPlayers
	_ = pl.save(pl.bannedPlayersFile, pl.BannedPlayers)
}

func (pl *AccessControl) UnbanIP(ip string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	newBannedIPs := make([]BanEntry, 0)
	for _, entry := range pl.BannedIPs {
		if entry.IP != ip {
			newBannedIPs = append(newBannedIPs, entry)
		}
	}
	pl.BannedIPs = newBannedIPs
	_ = pl.save(pl.bannedIPsFile, pl.BannedIPs)
}
