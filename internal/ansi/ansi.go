package ansi

const (
	Reset     = "\u001B[0m"
	Bold      = "\u001B[1m"
	Italic    = "\u001B[3m"
	Underline = "\u001B[4m"
	Strike    = "\u001B[9m"
	Red       = "\u001B[31m"
	Green     = "\u001B[32m"
	Yellow    = "\u001B[33m"
	Blue      = "\u001B[34m"
	Purple    = "\u001B[35m"
	Cyan      = "\u001B[36m"
	White     = "\u001B[37m"
	LightBlue = "\u001B[94m"
)

// MinecraftColors maps Minecraft text-component color names to ANSI escape codes.
var MinecraftColors = map[string]string{
	"black":        "\u001B[30m",
	"dark_blue":    "\u001B[34m",
	"dark_green":   "\u001B[32m",
	"dark_aqua":    "\u001B[36m",
	"dark_red":     "\u001B[31m",
	"dark_purple":  "\u001B[35m",
	"gold":         "\u001B[33m",
	"gray":         "\u001B[37m",
	"dark_gray":    "\u001B[90m",
	"blue":         "\u001B[94m",
	"green":        "\u001B[92m",
	"aqua":         "\u001B[96m",
	"red":          "\u001B[91m",
	"light_purple": "\u001B[95m",
	"yellow":       "\u001B[93m",
	"white":        "\u001B[97m",
}
