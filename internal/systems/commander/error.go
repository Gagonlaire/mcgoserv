package commander

import "fmt"

type ErrorCode int

const (
	UnknownCommand ErrorCode = iota
	IncompleteCommand
	InvalidArgument
)

type SyntaxError struct {
	Code     ErrorCode
	NodeName string
	Token    string
	Input    string
	Cursor   int
}

func (e *SyntaxError) String() string {
	switch e.Code {
	case UnknownCommand, IncompleteCommand:
		return "Unknown or incomplete command. See below for error"
	case InvalidArgument:
		return fmt.Sprintf("Unknown %s: %s", e.NodeName, e.Token)
	default:
		return "Syntax error"
	}
}

// todo: either return two strings (message and snippet) or add a method to get a text component
func (e *SyntaxError) Error() string {
	start := e.Cursor - 10
	prefix := "..."

	if start <= 0 {
		start = 0
		prefix = ""
	}

	snippet := e.Input[start:e.Cursor]
	msg := e.String()

	return fmt.Sprintf("%s\n%s%s<--[HERE]", msg, prefix, snippet)
}
