package parsers

import (
	"io"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type MessageType struct{}

var Message = MessageType{}

func (m MessageType) ID() int { return 20 } // minecraft:message

func (m MessageType) Parse(r *commander.CommandReader) (any, error) {
	if !r.CanRead() {
		return nil, commander.NewParsingErrorAt(
			tc.Text("Expected message"),
			r.Input(), r.Cursor(),
		)
	}

	start := r.Cursor()
	var selectors []mc.SelectorSpan
	for r.CanRead() {
		// this ensures the user is trying to input a selector and not just a message with an @ in it
		if r.Peek() == '@' && r.CanReadN(2) && mc.ValidSelectorVariable(r.Input()[r.Cursor()+1]) {
			selStart := r.Cursor() - start
			sel, err := parseSelector(r)
			if err != nil {
				return nil, err
			}
			selectors = append(selectors, mc.SelectorSpan{
				Start:    selStart,
				End:      r.Cursor() - start,
				Selector: sel,
			})
		} else {
			r.Skip()
		}
	}

	return &mc.ParsedMessage{
		Raw:       r.Input()[start:r.Cursor()],
		Selectors: selectors,
	}, nil
}

func (m MessageType) WriteTo(_ io.Writer) (int64, error) { return 0, nil }
