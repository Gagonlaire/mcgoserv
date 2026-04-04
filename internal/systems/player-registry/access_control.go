package player_registry

import (
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (pl *PlayerRegistry) IsWhitelisted(UUID uuid.UUID) bool {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	for _, entry := range pl.Whitelist {
		if entry.UUID == UUID.String() {
			return true
		}
	}
	return false
}

func (pl *PlayerRegistry) IsBanned(UUID uuid.UUID) (bool, *BanEntry) {
	pl.Mu.RLock()
	defer pl.Mu.RUnlock()

	now := time.Now()
	for _, entry := range pl.BannedPlayers {
		if entry.UUID == UUID.String() {
			if entry.Expires != "forever" && entry.Expires != "" {
				expireTime, err := time.Parse("2006-01-02 15:04:05 -0700", entry.Expires)
				if err == nil && now.After(expireTime) {
					// todo: check if entry should be removed from the list
					continue
				}
			}
			return true, &entry
		}
	}
	return false, nil
}

func (pl *PlayerRegistry) IsIPBanned(ip string) (bool, *BanEntry) {
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

func (pl *PlayerRegistry) AddWhitelist(UUID uuid.UUID, name string) {
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

func (pl *PlayerRegistry) RemoveWhitelist(name string) {
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

func (pl *PlayerRegistry) RemoveWhitelistByUUID(uuidStr string) (WhitelistEntry, bool) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	for i, entry := range pl.Whitelist {
		if entry.UUID == uuidStr {
			removed := entry
			pl.Whitelist = append(pl.Whitelist[:i], pl.Whitelist[i+1:]...)
			_ = pl.save(pl.whitelistFile, pl.Whitelist)
			return removed, true
		}
	}
	return WhitelistEntry{}, false
}

func (pl *PlayerRegistry) RemoveWhitelistByName(name string, caseSensitive bool) (WhitelistEntry, bool) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	for i, entry := range pl.Whitelist {
		var match bool
		if caseSensitive {
			match = entry.Name == name
		} else {
			match = strings.EqualFold(entry.Name, name)
		}
		if match {
			removed := entry
			pl.Whitelist = append(pl.Whitelist[:i], pl.Whitelist[i+1:]...)
			_ = pl.save(pl.whitelistFile, pl.Whitelist)
			return removed, true
		}
	}
	return WhitelistEntry{}, false
}

func (pl *PlayerRegistry) ReconcileWhitelistName(UUID uuid.UUID, name string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	uuidStr := UUID.String()
	for i, entry := range pl.Whitelist {
		if entry.UUID == uuidStr && entry.Name == "Unknown" {
			pl.Whitelist[i].Name = name
			_ = pl.save(pl.whitelistFile, pl.Whitelist)
			return
		}
	}
}

func (pl *PlayerRegistry) Ban(UUID uuid.UUID, name, source, reason, expires string) {
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

func (pl *PlayerRegistry) BanIP(ip, source, reason, expires string) {
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

func (pl *PlayerRegistry) UnbanByUUID(uuidStr string) {
	pl.Mu.Lock()
	defer pl.Mu.Unlock()

	newBannedPlayers := make([]BanEntry, 0)
	for _, entry := range pl.BannedPlayers {
		if entry.UUID != uuidStr {
			newBannedPlayers = append(newBannedPlayers, entry)
		}
	}
	pl.BannedPlayers = newBannedPlayers
	_ = pl.save(pl.bannedPlayersFile, pl.BannedPlayers)
}

func (pl *PlayerRegistry) Unban(name string) {
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

func (pl *PlayerRegistry) UnbanIP(ip string) {
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
