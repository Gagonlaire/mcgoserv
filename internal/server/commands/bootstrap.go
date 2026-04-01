package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/server"
)

func RegisterAll(s *server.Server) {
	registerCommon(s)
	registerList(s)
	registerKick(s)
	registerBan(s)
	registerWhitelist(s)
}
