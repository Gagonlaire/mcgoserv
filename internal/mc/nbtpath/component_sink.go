package nbtpath

import (
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Tnze/go-mc/nbt"
)

const (
	colorKey    = tc.ColorAqua
	colorNum    = tc.ColorGold
	colorType   = tc.ColorRed
	colorString = tc.ColorGreen
	colorPunct  = tc.ColorWhite
)

// ComponentSink renders SNBT into a styled *tc.TextComponent.
type ComponentSink struct {
	root *tc.TextComponent
}

func NewComponentSink() *ComponentSink {
	return &ComponentSink{root: tc.Container()}
}

func (cs *ComponentSink) Result() *tc.TextComponent { return cs.root }

func (cs *ComponentSink) Punct(s string)  { cs.root.AddExtra(tc.Text(s).SetColor(colorPunct)) }
func (cs *ComponentSink) Key(s string)    { cs.root.AddExtra(tc.Text(s).SetColor(colorKey)) }
func (cs *ComponentSink) String(s string) { cs.root.AddExtra(tc.Text(s).SetColor(colorString)) }
func (cs *ComponentSink) Number(s string) { cs.root.AddExtra(tc.Text(s).SetColor(colorNum)) }
func (cs *ComponentSink) Type(s string)   { cs.root.AddExtra(tc.Text(s).SetColor(colorType)) }

// FormatSNBTComponent walks v and produces a colored text component.
func FormatSNBTComponent(v any) *tc.TextComponent {
	sink := NewComponentSink()
	WriteSNBT(sink, v)
	return sink.Result()
}

// SNBTToComponent parses an SNBT string and renders it as a colored component.
func SNBTToComponent(snbt nbt.StringifiedMessage) (*tc.TextComponent, error) {
	v, err := SNBTToValue(snbt)
	if err != nil {
		return nil, err
	}
	return FormatSNBTComponent(v), nil
}
