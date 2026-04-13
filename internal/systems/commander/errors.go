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

func formatError(component tc.Component, input string, cursor int, hasCursor bool) tc.Component {
	var context tc.Component
	if hasCursor {
		c := max(0, min(cursor, len(input)))
		before := input[:c]
		after := input[c:]
		if len(before) > 10 {
			before = "..." + before[len(before)-10:]
		}
		context = tc.Container(
			tc.Text("\n"+before).SetColor(tc.ColorGray),
			tc.Text(after).SetUnderlined(true),
			tc.Translatable(mcdata.CommandContextHere),
		).SuggestCommand("/" + input)
	} else {
		preview := input
		if len(preview) > 10 {
			preview = "..." + preview[len(preview)-10:]
		}
		context = tc.Container(
			tc.Text("\n"+preview).SetColor(tc.ColorGray),
			tc.Translatable(mcdata.CommandContextHere),
		).SuggestCommand("/" + input)
	}
	return tc.Container(component, context).SetColor(tc.ColorRed)
}

type CommandParsingError struct {
	component tc.Component
	input     string
	cursor    int
	hasCursor bool
}

func NewParsingError(component tc.Component, input string) *CommandParsingError {
	return &CommandParsingError{component: component, input: input}
}

func NewParsingErrorAt(component tc.Component, input string, cursor int) *CommandParsingError {
	return &CommandParsingError{component: component, input: input, cursor: cursor, hasCursor: true}
}

func (e *CommandParsingError) Error() string { return e.component.String() }

func (e *CommandParsingError) ToComponent() tc.Component {
	return formatError(e.component, e.input, e.cursor, e.hasCursor)
}

type CommandExecutionError struct {
	component tc.Component
	input     string
	cursor    int
	hasCursor bool
}

func NewExecutionError(component tc.Component, input string) *CommandExecutionError {
	return &CommandExecutionError{component: component, input: input}
}

func NewExecutionErrorAt(component tc.Component, input string, cursor int) *CommandExecutionError {
	return &CommandExecutionError{component: component, input: input, cursor: cursor, hasCursor: true}
}

func (e *CommandExecutionError) Error() string { return e.component.String() }

func (e *CommandExecutionError) ToComponent() tc.Component {
	return formatError(e.component, e.input, e.cursor, e.hasCursor)
}
