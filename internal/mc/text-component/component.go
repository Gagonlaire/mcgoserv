package text_component

import (
	"encoding/json"
	"io"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Tnze/go-mc/nbt"
)

type Component interface {
	WriteTo(w io.Writer) (int64, error)
	ToJSON() []byte
	String() string

	AnsiString() string
	AnsiLines() []string
	renderAnsi(parentStyle string) string
}

type Base[T any] struct {
	self  T
	Type  string      `nbt:"type" json:"type"`
	Extra []Component `nbt:"extra,omitempty" json:"extra,omitempty"`
	*Formatting
	*Interactivity
}

func (b *Base[T]) WriteTo(w io.Writer) (int64, error) {
	encoder := nbt.NewEncoder(w)
	encoder.NetworkFormat(true)
	err := encoder.Encode(b.self, "")

	if err != nil {
		return 0, err
	}
	return 0, nil
}

func (b *Base[T]) ToJSON() []byte {
	data, err := json.Marshal(b.self)

	if err != nil {
		return nil
	}
	return data
}

func (b *Base[T]) AddExtra(children ...Component) T {
	b.Extra = append(b.Extra, children...)
	return b.self
}

type TextComponent struct {
	Base[*TextComponent]
	Text string `nbt:"text" json:"text"`
}

func (t *TextComponent) String() string {
	var sb strings.Builder
	sb.WriteString(t.Text)
	for _, extra := range t.Extra {
		sb.WriteString(extra.String())
	}
	return sb.String()
}

func (t *TextComponent) AnsiString() string {
	return t.renderAnsi("")
}

func (t *TextComponent) renderAnsi(parentStyle string) string {
	return t.buildAnsiString(t.Text, parentStyle)
}

func (t *TextComponent) AnsiLines() []string {
	return splitAnsiString(t.AnsiString())
}

type TranslatableComponent struct {
	Base[*TranslatableComponent]
	Translate string      `nbt:"translate" json:"translate"`
	Fallback  string      `nbt:"fallback,omitempty" json:"fallback,omitempty"`
	With      []Component `nbt:"with,omitempty" json:"with,omitempty"`
}

func (t *TranslatableComponent) String() string {
	args := make([]interface{}, len(t.With))
	for i, comp := range t.With {
		args[i] = comp.String()
	}

	translated := mcdata.TranslationKey(t.Translate).Format(args...)

	var sb strings.Builder
	sb.WriteString(translated)
	for _, extra := range t.Extra {
		sb.WriteString(extra.String())
	}
	return sb.String()
}

func (t *TranslatableComponent) AnsiString() string {
	return t.renderAnsi("")
}

func (t *TranslatableComponent) renderAnsi(parentStyle string) string {
	args := make([]interface{}, len(t.With))

	for i, comp := range t.With {
		args[i] = comp.renderAnsi(parentStyle)
	}
	translated := mcdata.TranslationKey(t.Translate).Format(args...)

	return t.buildAnsiString(translated, parentStyle)
}

func (t *TranslatableComponent) AnsiLines() []string {
	return splitAnsiString(t.AnsiString())
}

func Text(text string) *TextComponent {
	t := &TextComponent{
		Text: text,
		Base: Base[*TextComponent]{
			Type:          "text",
			Formatting:    &Formatting{},
			Interactivity: &Interactivity{},
		},
	}
	t.Base.self = t

	return t
}

func Translatable(translation mcdata.TranslationKey, with ...Component) *TranslatableComponent {
	t := &TranslatableComponent{
		Translate: string(translation),
		Base: Base[*TranslatableComponent]{
			Type:          "translatable",
			Formatting:    &Formatting{},
			Interactivity: &Interactivity{},
		},
		With: with,
	}
	t.Base.self = t

	return t
}
