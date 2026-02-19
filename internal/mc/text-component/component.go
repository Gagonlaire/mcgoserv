package text_component

import (
	"encoding/json"
	"io"

	"github.com/Tnze/go-mc/nbt"
)

type Component interface {
	isComponent() // lock interface
}

type Base[T any] struct {
	self  T
	Type  string      `nbt:"type" json:"type"`
	Extra []Component `nbt:"extra,omitempty" json:"extra,omitempty"`
	*Formatting
	*Interactivity
}

func (b *Base[T]) isComponent() {}

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
	data, err := json.Marshal(b)

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

type TranslatableComponent struct {
	Base[*TranslatableComponent]
	Translate string      `nbt:"translate" json:"translate"`
	Fallback  string      `nbt:"fallback,omitempty" json:"fallback,omitempty"`
	With      []Component `nbt:"with,omitempty" json:"with,omitempty"`
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

func Translatable(translate string, with ...Component) *TranslatableComponent {
	t := &TranslatableComponent{
		Translate: translate,
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
