package commander

import (
	"context"
)

type CommandSource struct {
	Server          any
	PermissionLevel int
	Entity          any
	Position        [3]float64
	Rotation        [2]float32
	SendMessage     func(msg any)
}

func (s *CommandSource) HasPermission(level int) bool {
	return s.PermissionLevel >= level
}

func (s *CommandSource) Clone() *CommandSource {
	c := *s
	return &c
}

type CommandResult struct {
	Success int
	Result  int
}

type RedirectModifier func(ctx context.Context, src *CommandSource) ([]*CommandSource, error)
