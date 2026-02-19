package text_component

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

type Formatting struct {
	Color         string `nbt:"color,omitempty" json:"color,omitempty"`
	Font          string `nbt:"font,omitempty" json:"font,omitempty"`
	Bold          bool   `nbt:"bold,omitempty" json:"bold,omitempty"`
	Italic        bool   `nbt:"italic,omitempty" json:"italic,omitempty"`
	Underlined    bool   `nbt:"underlined,omitempty" json:"underlined,omitempty"`
	Strikethrough bool   `nbt:"strikethrough,omitempty" json:"strikethrough,omitempty"`
	Obfuscated    bool   `nbt:"obfuscated,omitempty" json:"obfuscated,omitempty"`
	ShadowColor   any    `nbt:"shadow_color,omitempty" json:"shadow_color,omitempty"`
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
