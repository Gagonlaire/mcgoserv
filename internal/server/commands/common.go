package commands

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/buildinfo"
	"github.com/Gagonlaire/mcgoserv/internal/logger"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

const repoURL = "https://github.com/Gagonlaire/mcgoserv"

func versionLine(label, value string) tc.Component {
	return tc.Container(
		tc.Text(label).SetColor(tc.ColorGold),
		tc.Text(value).SetColor(tc.ColorWhite),
	)
}

func seriesComponent() tc.Component {
	branch := buildinfo.BranchValue()
	branchComp := tc.Text(branch).SetColor(tc.ColorWhite)

	if branch != "unknown" {
		branchURL := repoURL + "/tree/" + branch
		branchComp = branchComp.
			OpenURL(branchURL).
			ShowText(tc.Text("Open branch on GitHub"))
	}

	return tc.Container(
		tc.Text("series: ").SetColor(tc.ColorGold),
		branchComp,
	)
}

func stableComponent() tc.Component {
	if buildinfo.IsStable() {
		return tc.Text("yes").SetColor(tc.ColorGreen)
	}
	return tc.Text("no").SetColor(tc.ColorRed)
}

func buildTimeValue() string {
	if buildinfo.BuildTime == "" {
		return "unknown"
	}
	return buildinfo.BuildTime
}

func registerCommon(s *server.Server) {
	s.Commander.Register(
		Literal("stop").Requires(4).Executes(func(cc *CommandContext) (*CommandResult, error) {
			logger.Component(logger.INFO, tc.Text("Stopping the server"))
			s.Stop()

			return &CommandResult{Success: 1, Result: 0}, nil
		}),

		Literal("version").Executes(func(cc *CommandContext) (*CommandResult, error) {
			header := tc.Container(
				tc.Text("McGoServ").SetColor(tc.ColorGreen).SetBold(true),
				tc.Text(" — ").SetColor(tc.ColorDarkGray),
				tc.Text("A Minecraft server written in Go").SetColor(tc.ColorGray).SetItalic(true),
			)
			link := tc.Container(
				tc.Text(repoURL).
					SetColor(tc.ColorAqua).SetUnderlined(true).
					OpenURL(repoURL).
					ShowText(tc.Text("Open repository on GitHub")),
			)

			cc.SendMessage(tc.Container(
				tc.Text("\n"),
				header,
				tc.Text("\n"),
				link,
				tc.Text("\n\n"),
				versionLine("version: ", mcdata.GameVersion),
				tc.Text("\n"),
				versionLine("protocol: ", fmt.Sprintf("%d (0x%X)", mcdata.ProtocolVersion, mcdata.ProtocolVersion)),
				tc.Text("\n"),
				seriesComponent(),
				tc.Text("\n"),
				versionLine("build_time: ", buildTimeValue()),
				tc.Text("\n"),
				tc.Container(
					tc.Text("stable: ").SetColor(tc.ColorGold),
					stableComponent(),
				),
				tc.Text("\n"),
			))

			return &CommandResult{Success: 1, Result: 0}, nil
		}),
	)
}
