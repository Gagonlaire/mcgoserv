package text_component

import (
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal"
)

const (
	ColorBlack       = "black"
	ColorDarkBlue    = "dark_blue"
	ColorDarkGreen   = "dark_green"
	ColorDarkAqua    = "dark_aqua"
	ColorDarkRed     = "dark_red"
	ColorDarkPurple  = "dark_purple"
	ColorGold        = "gold"
	ColorGray        = "gray"
	ColorDarkGray    = "dark_gray"
	ColorBlue        = "blue"
	ColorGreen       = "green"
	ColorAqua        = "aqua"
	ColorRed         = "red"
	ColorLightPurple = "light_purple"
	ColorYellow      = "yellow"
	ColorWhite       = "white"
)

var AnsiColors = map[string]string{
	ColorBlack:       "\u001B[30m",
	ColorDarkBlue:    "\u001B[34m",
	ColorDarkGreen:   "\u001B[32m",
	ColorDarkAqua:    "\u001B[36m",
	ColorDarkRed:     "\u001B[31m",
	ColorDarkPurple:  "\u001B[35m",
	ColorGold:        "\u001B[33m",
	ColorGray:        "\u001B[37m",
	ColorDarkGray:    "\u001B[90m",
	ColorBlue:        "\u001B[94m",
	ColorGreen:       "\u001B[92m",
	ColorAqua:        "\u001B[96m",
	ColorRed:         "\u001B[91m",
	ColorLightPurple: "\u001B[95m",
	ColorYellow:      "\u001B[93m",
	ColorWhite:       "\u001B[97m",
}

type Formatting struct {
	ShadowColor   any    `nbt:"shadow_color,omitempty" json:"shadow_color,omitempty"`
	Color         string `nbt:"color,omitempty" json:"color,omitempty"`
	Font          string `nbt:"font,omitempty" json:"font,omitempty"`
	Bold          bool   `nbt:"bold,omitempty" json:"bold,omitempty"`
	Italic        bool   `nbt:"italic,omitempty" json:"italic,omitempty"`
	Underlined    bool   `nbt:"underlined,omitempty" json:"underlined,omitempty"`
	Strikethrough bool   `nbt:"strikethrough,omitempty" json:"strikethrough,omitempty"`
	Obfuscated    bool   `nbt:"obfuscated,omitempty" json:"obfuscated,omitempty"`
}

func (b *Base[T]) SetColor(color string) T {
	b.Color = color
	return b.self
}

func (b *Base[T]) SetFont(font string) T {
	b.Font = font
	return b.self
}

func (b *Base[T]) SetBold(bold bool) T {
	b.Bold = bold
	return b.self
}

func (b *Base[T]) SetItalic(italic bool) T {
	b.Italic = italic
	return b.self
}

func (b *Base[T]) SetUnderlined(underlined bool) T {
	b.Underlined = underlined
	return b.self
}

func (b *Base[T]) SetStrikethrough(strikethrough bool) T {
	b.Strikethrough = strikethrough
	return b.self
}

func (b *Base[T]) SetObfuscated(obfuscated bool) T {
	b.Obfuscated = obfuscated
	return b.self
}

// SetShadowColor shadowColor must either be a packed int32 or a list of 4 floats representing RGBA values.
func (b *Base[T]) SetShadowColor(shadowColor any) T {
	b.ShadowColor = shadowColor
	return b.self
}

// buildAnsiString add ANSI codes for a component representation, keeping the parent styles
func (b *Base[T]) buildAnsiString(content string, parentStyle string) string {
	var sb strings.Builder
	var myStyleBuilder strings.Builder

	if b.Formatting != nil {
		if code, ok := AnsiColors[b.Formatting.Color]; ok {
			myStyleBuilder.WriteString(code)
		}
		if b.Formatting.Bold {
			myStyleBuilder.WriteString(internal.AnsiBold)
		}
		if b.Formatting.Italic {
			myStyleBuilder.WriteString(internal.AnsiItalic)
		}
		if b.Formatting.Underlined {
			myStyleBuilder.WriteString(internal.AnsiUnderline)
		}
		if b.Formatting.Strikethrough {
			myStyleBuilder.WriteString(internal.AnsiStrike)
		}
	}

	myStyle := myStyleBuilder.String()
	effectiveStyle := parentStyle + myStyle

	sb.WriteString(myStyle)
	sb.WriteString(content)
	for _, extra := range b.Extra {
		sb.WriteString(extra.renderAnsi(effectiveStyle))
	}

	if myStyle != "" {
		sb.WriteString(internal.AnsiReset)
		sb.WriteString(parentStyle)
	}

	return sb.String()
}

// splitAnsiString split in multiple lines for logging, while preserving style to not mess logger formatting
func splitAnsiString(str string) []string {
	var lines []string
	var currentLine strings.Builder
	var activeStyle strings.Builder

	inEscape := false
	escapeBuf := strings.Builder{}
	for _, r := range str {
		if inEscape {
			escapeBuf.WriteRune(r)
			if r == 'm' {
				code := escapeBuf.String()
				inEscape = false
				escapeBuf.Reset()

				currentLine.WriteString(code)

				if code == internal.AnsiReset {
					activeStyle.Reset()
				} else {
					activeStyle.WriteString(code)
				}
			}
			continue
		}

		if r == '\x1b' {
			inEscape = true
			escapeBuf.WriteRune(r)
			continue
		}

		if r == '\n' {
			currentLine.WriteString(internal.AnsiReset)
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(activeStyle.String())
			continue
		}

		currentLine.WriteRune(r)
	}

	if currentLine.Len() > 0 {
		currentLine.WriteString(internal.AnsiReset)
		lines = append(lines, currentLine.String())
	}

	return lines
}
