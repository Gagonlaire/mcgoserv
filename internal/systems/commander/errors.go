package commander

import (
	"fmt"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
)

type CommandParseError struct {
	Component tc.Component
	Input     string
	Cursor    int
}

func NewParseError(component tc.Component, input string, cursor int) *CommandParseError {
	return &CommandParseError{
		Component: component,
		Input:     input,
		Cursor:    cursor,
	}
}

func (e *CommandParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Cursor, e.Component.String())
}

func (e *CommandParseError) ToComponent() tc.Component {
	context := e.Input
	suffix := ""
	if e.Cursor >= 0 && e.Cursor <= len(e.Input) {
		context = e.Input[:e.Cursor]
		suffix = e.Input[e.Cursor:]
	}

	msg := tc.Container(e.Component).SetColor(tc.ColorRed)
	if len(context) > 0 || len(suffix) > 0 {
		msg.AddExtra(
			tc.Text("\n"+context).SetColor(tc.ColorGray),
			tc.Text(suffix).SetUnderlined(true).SetColor(tc.ColorRed),
			tc.Text("<--[HERE]").SetColor(tc.ColorRed).SetItalic(true),
		)
	}
	return msg
}

type CommandExecutionError struct {
	Component tc.Component
	Cause     error
}

func NewCommandError(component tc.Component) *CommandExecutionError {
	return &CommandExecutionError{Component: component}
}

func (e *CommandExecutionError) Error() string {
	msg := e.Component.String()
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", msg, e.Cause)
	}
	return msg
}

func (e *CommandExecutionError) ToComponent() tc.Component {
	return tc.Container(e.Component).SetColor(tc.ColorRed)
}
