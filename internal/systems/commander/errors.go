package commander

import (
	"errors"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

type CommandError interface {
	error
	ToComponent() tc.Component
}

func AsCommandError(err error) CommandError {
	if ce, ok := errors.AsType[CommandError](err); ok {
		return ce
	}
	panic("commander: trying to convert non-command error to CommandError")
}

func formatError(component tc.Component, input string, cursor int) tc.Component {
	c := max(0, min(cursor, len(input)))
	before := input[:c]
	after := input[c:]
	if len(before) > 10 {
		before = "..." + before[len(before)-10:]
	}
	context := tc.Container(
		tc.Text("\n"+before).SetColor(tc.ColorGray),
		tc.Text(after).SetUnderlined(true),
		tc.Translatable(mcdata.CommandContextHere),
	).SuggestCommand("/" + input)
	return tc.Container(component, context).SetColor(tc.ColorRed)
}

type CommandParsingError struct {
	component tc.Component
	input     string
	cursor    int
}

func NewParsingError(r *CommandReader, component tc.Component) *CommandParsingError {
	return &CommandParsingError{component: component, input: r.Input(), cursor: r.Cursor()}
}

func NewParsingErrorAt(r *CommandReader, component tc.Component, cursor int) *CommandParsingError {
	return &CommandParsingError{component: component, input: r.Input(), cursor: cursor}
}

func (e *CommandParsingError) Error() string { return e.component.String() }

func (e *CommandParsingError) Cursor() int { return e.cursor }

func (e *CommandParsingError) Input() string { return e.input }

func (e *CommandParsingError) Component() tc.Component { return e.component }

func (e *CommandParsingError) ToComponent() tc.Component {
	return formatError(e.component, e.input, e.cursor)
}

type CommandExecutionError struct {
	component tc.Component
	input     string
	cursor    int
}

func NewExecutionError(r *CommandReader, component tc.Component) *CommandExecutionError {
	return &CommandExecutionError{component: component, input: r.Input(), cursor: r.Cursor()}
}

func NewExecutionErrorAt(r *CommandReader, component tc.Component, cursor int) *CommandExecutionError {
	return &CommandExecutionError{component: component, input: r.Input(), cursor: cursor}
}

func NewExecutionErrorFromParsing(e *CommandParsingError) *CommandExecutionError {
	return &CommandExecutionError{component: e.component, input: e.input, cursor: e.cursor}
}

func (e *CommandExecutionError) Error() string { return e.component.String() }

func (e *CommandExecutionError) Cursor() int { return e.cursor }

func (e *CommandExecutionError) ToComponent() tc.Component {
	return formatError(e.component, e.input, e.cursor)
}
