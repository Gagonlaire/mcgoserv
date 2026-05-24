package commands

import (
	"sort"
	"strings"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/server"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander/parsers"
)

func registerHelp(s *server.Server) {
	s.Commander.RegisterBuilders(func() {
		Build("/help [<command>]", parsers.String.Behavior(parsers.GreedyPhrase)).
			ExecutesEach(helpAll(s), helpFor(s))
	})
}

func helpAll(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		lines := usageLines(s, s.Commander.Root, "", cc.Source.PermissionLevel)
		emitUsageLines(cc, lines)
		return &CommandResult{Success: 1, Result: len(lines)}, nil
	}
}

func helpFor(s *server.Server) Command {
	return func(cc *CommandContext) (*CommandResult, error) {
		raw := strings.TrimSpace(cc.Args.GetString("command"))
		tokens := strings.Fields(raw)
		if len(tokens) == 0 {
			return helpAll(s)(cc)
		}

		node := s.Commander.Root
		for _, tok := range tokens {
			next := findChildByName(node, tok, cc.Source.PermissionLevel)
			if next == nil {
				cc.SendMessage(tc.Translatable(mcdata.CommandsHelpFailed).SetColor(tc.ColorRed))
				return &CommandResult{Success: 0}, nil
			}
			node = next
		}

		if node.Redirect != nil {
			node = node.Redirect
		}

		lines := usageLines(s, node, strings.Join(tokens, " "), cc.Source.PermissionLevel)
		if len(lines) == 0 {
			cc.SendMessage(tc.Translatable(mcdata.CommandsHelpFailed).SetColor(tc.ColorRed))
			return &CommandResult{Success: 0}, nil
		}
		emitUsageLines(cc, lines)
		return &CommandResult{Success: 1, Result: len(lines)}, nil
	}
}

func usageLines(s *server.Server, node *Node, pathPrefix string, perm int) []string {
	if node.Description != "" {
		return []string{node.Description}
	}
	lines := s.Commander.UsageLines(node, pathPrefix, perm)
	sort.Strings(lines)
	return lines
}

func emitUsageLines(cc *CommandContext, lines []string) {
	for _, line := range lines {
		cc.SendMessage(tc.Text(line).SuggestCommand(line))
	}
}

func findChildByName(node *Node, name string, perm int) *Node {
	for _, c := range node.Children {
		if c.PermissionLevel > 0 && c.PermissionLevel > perm {
			continue
		}
		if c.Kind == LiteralNode && c.Name == name {
			return c
		}
	}
	return nil
}
