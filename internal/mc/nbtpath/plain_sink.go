package nbtpath

import (
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// PlainSink renders SNBT into a plain string with no styling.
type PlainSink struct {
	b strings.Builder
}

func (p *PlainSink) Punct(s string)  { p.b.WriteString(s) }
func (p *PlainSink) Key(s string)    { p.b.WriteString(s) }
func (p *PlainSink) String(s string) { p.b.WriteString(s) }
func (p *PlainSink) Number(s string) { p.b.WriteString(s) }
func (p *PlainSink) Type(s string)   { p.b.WriteString(s) }

func (p *PlainSink) Result() string { return p.b.String() }

// FormatSNBT renders v as a plain SNBT string.
func FormatSNBT(v any) nbt.StringifiedMessage {
	var sink PlainSink
	WriteSNBT(&sink, v)
	return nbt.StringifiedMessage(sink.Result())
}
