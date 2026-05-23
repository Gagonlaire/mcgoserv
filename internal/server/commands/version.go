package commands

import (
	"fmt"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

const repoURL = "https://github.com/Gagonlaire/mcgoserv"

// Build metadata injected via -ldflags at build time. See Makefile.
var (
	BuildTime string
	Stable    string
	Branch    string
)

func registerVersion(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/version").Executes(func(cc *CommandContext) (*CommandResult, error) {
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

			// todo: implement a fmt helper for tc
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
		})
	})
}

func isStable() bool {
	return Stable == "true"
}

func branchValue() string {
	if Branch == "" {
		return "unknown"
	}
	return Branch
}

func buildTimeValue() string {
	if BuildTime == "" {
		return "unknown"
	}
	return BuildTime
}

func versionLine(label, value string) tc.Component {
	return tc.Container(
		tc.Text(label).SetColor(tc.ColorGold),
		tc.Text(value).SetColor(tc.ColorWhite),
	)
}

func seriesComponent() tc.Component {
	branch := branchValue()
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
	if isStable() {
		return tc.Text("yes").SetColor(tc.ColorGreen)
	}
	return tc.Text("no").SetColor(tc.ColorRed)
}
