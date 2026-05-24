package commander

import (
	"context"

	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
)

type EntityAnchor uint8

const (
	AnchorFeet EntityAnchor = iota
	AnchorEyes
)

type ResultConsumer interface {
	OnResult(src *CommandSource, success bool, result int)
}

type CommandSource struct {
	Server          any
	Entity          entity.Entity
	SendMessage     func(msg any)
	ResultConsumer  ResultConsumer
	Dimension       proto.Identifier
	Position        [3]float64
	Rotation        [2]float32
	PermissionLevel int
	Anchor          EntityAnchor
	Silent          bool
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

// RedirectModifier produces the source set to use when execution crosses a redirecting node.
type RedirectModifier func(ctx context.Context, src *CommandSource) ([]*CommandSource, error)

func (s *CommandSource) HasPermission(level int) bool {
	return s.PermissionLevel >= level
}

func (s *CommandSource) WithEntity(e entity.Entity) *CommandSource {
	cp := *s
	cp.Entity = e
	return &cp
}

func (s *CommandSource) WithPosition(p [3]float64) *CommandSource {
	cp := *s
	cp.Position = p
	return &cp
}

func (s *CommandSource) WithRotation(rot [2]float32) *CommandSource {
	cp := *s
	cp.Rotation = rot
	return &cp
}

func (s *CommandSource) WithAnchor(a EntityAnchor) *CommandSource {
	cp := *s
	cp.Anchor = a
	return &cp
}

func (s *CommandSource) WithDimension(d proto.Identifier) *CommandSource {
	cp := *s
	cp.Dimension = d
	return &cp
}

func (s *CommandSource) WithSilent(silent bool) *CommandSource {
	cp := *s
	cp.Silent = silent
	return &cp
}

func (s *CommandSource) WithResultConsumer(c ResultConsumer) *CommandSource {
	cp := *s
	cp.ResultConsumer = c
	return &cp
}

func (sd *SignedData) GetArgSignature(name string) []byte {
	if sd == nil {
		return nil
	}
	return sd.ArgSignatures[name]
}

func (cc *CommandContext) SendMessage(msg any) {
	if cc.Source.Silent {
		return
	}
	if cc.Source.SendMessage != nil {
		cc.Source.SendMessage(msg)
	}
}
