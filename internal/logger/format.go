package logger

import "fmt"

import "github.com/Gagonlaire/mcgoserv/internal"

func Identity(v any) string {
	return internal.ColorCyan + fmt.Sprint(v) + internal.AnsiReset
}

func Network(v any) string {
	return internal.ColorPurple + fmt.Sprint(v) + internal.AnsiReset
}

func Value(v any) string {
	return internal.ColorGreen + internal.AnsiBold + fmt.Sprint(v) + internal.AnsiReset
}
