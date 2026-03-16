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

type SignedData struct {
	ArgSignatures      map[string][]byte
	LastSeenSignatures [][]byte
	Timestamp          int64
	Salt               int64
}

type CommandContext struct {
	Ctx    context.Context
	Source *CommandSource
	Args   ParsedArgs
	Signed *SignedData
}

type CommandResult struct {
	Success int
	Result  int
}

type RedirectModifier func(ctx context.Context, src *CommandSource) ([]*CommandSource, error)

func (s *CommandSource) HasPermission(level int) bool {
	return s.PermissionLevel >= level
}

func (sd *SignedData) GetArgSignature(name string) []byte {
	if sd == nil {
		return nil
	}
	return sd.ArgSignatures[name]
}

func (cc *CommandContext) SendMessage(msg any) {
	if cc.Source.SendMessage != nil {
		cc.Source.SendMessage(msg)
	}
}
