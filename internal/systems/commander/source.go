package commander

import (
	"context"
)

type CommandSource struct {
	Server          any
	Entity          any
	SendMessage     func(msg any)
	Position        [3]float64
	PermissionLevel int
	Rotation        [2]float32
}

func (s *CommandSource) HasPermission(level int) bool {
	return s.PermissionLevel >= level
}

func (s *CommandSource) Clone() *CommandSource {
	c := *s
	return &c
}

type CommandContext struct {
	Ctx    context.Context
	Source *CommandSource
	Args   ParsedArgs
}

func (cc *CommandContext) SendMessage(msg any) {
	if cc.Source.SendMessage != nil {
		cc.Source.SendMessage(msg)
	}
}

type CommandResult struct {
	Success int
	Result  int
}

type RedirectModifier func(ctx context.Context, src *CommandSource) ([]*CommandSource, error)
