package logger

import (
	"fmt"

	"github.com/Gagonlaire/mcgoserv/internal/ansi"
)

func Identity(v any) string {
	return ansi.Cyan + fmt.Sprint(v) + ansi.Reset
}

func Network(v any) string {
	return ansi.Purple + fmt.Sprint(v) + ansi.Reset
}

func Value(v any) string {
	return ansi.Green + ansi.Bold + fmt.Sprint(v) + ansi.Reset
}

func FmtWarn(v any) string {
	return ansi.Yellow + fmt.Sprint(v) + ansi.Reset
}
