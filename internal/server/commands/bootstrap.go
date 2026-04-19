package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
)

// RegisterAll https://minecraft.wiki/w/Commands#List_of_commands
func RegisterAll(s *server.Server) {
	registerBan(s)
	registerBanIP(s)
	registerData(s)
	registerDeop(s)
	registerKick(s)
	registerKill(s)
	registerList(s)
	registerMsg(s)
	registerOp(s)
	registerPardon(s)
	registerPardonIP(s)
	registerSay(s)
	registerStop(s)
	registerSummon(s)
	registerTeamMsg(s)
	registerTellRaw(s)
	registerVersion(s)
	registerWhitelist(s)
}
