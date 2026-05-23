package commands

import (
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func registerStop(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/stop").Requires(4).Executes(func(cc *CommandContext) (*CommandResult, error) {
			logger.Component(logger.INFO, tc.Text("Stopping the server"))
			s.Stop()

			return &CommandResult{Success: 1, Result: 0}, nil
		})
	})
}
