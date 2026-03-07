package parsers

import (
	"io"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type MessageType struct{}

type ParsedMessage struct {
	Format    string
	Selectors []*mc.Selector
}

var Message = MessageType{}

func (m MessageType) ID() int { return 20 } // minecraft:message

func (m MessageType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Text("Expected message"),
			r.Input(), r.Cursor(),
		)
	}

	var buf strings.Builder
	var selectors []*mc.Selector
	for r.CanRead() {
		// this ensures the user is trying to input a selector and not just a message with an @ in it
		if r.Peek() == '@' && r.CanReadN(2) && mc.ValidSelectorVariable(r.Input()[r.Cursor()+1]) {
			sel, err := parseSelector(r)
			if err != nil {
				return nil, err
			}
			selectors = append(selectors, sel)
			buf.WriteString("%s")
		} else {
			buf.WriteByte(r.Read())
		}
	}

	return &ParsedMessage{
		Format:    buf.String(),
		Selectors: selectors,
	}, nil
}

func (m MessageType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }
