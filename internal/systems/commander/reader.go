package commander

import (
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
)

type CommandReader struct {
	input  string
	cursor int
}

func NewCommandReader(input string) *CommandReader {
	return &CommandReader{input: input}
}

func (r *CommandReader) Input() string { return r.input }

func (r *CommandReader) Cursor() int { return r.cursor }

func (r *CommandReader) SetCursor(pos int) { r.cursor = pos }

func (r *CommandReader) TotalLength() int { return len(r.input) }

func (r *CommandReader) RemainingLength() int { return len(r.input) - r.cursor }

func (r *CommandReader) CanRead() bool { return r.cursor < len(r.input) }

func (r *CommandReader) CanReadN(n int) bool { return r.cursor+n <= len(r.input) }

func (r *CommandReader) Peek() byte { return r.input[r.cursor] }

func (r *CommandReader) Read() byte {
	b := r.input[r.cursor]
	r.cursor++
	return b
}

func (r *CommandReader) Skip() { r.cursor++ }

func (r *CommandReader) GetRead() string { return r.input[:r.cursor] }

func (r *CommandReader) GetRemaining() string { return r.input[r.cursor:] }

func (r *CommandReader) SkipWhitespace() {
	for r.CanRead() && r.Peek() == ' ' {
		r.Skip()
	}
}

func (r *CommandReader) ReadUnquotedString() string {
	start := r.cursor
	for r.CanRead() && IsAllowedInUnquotedString(r.Peek()) {
		r.Skip()
	}
	return r.input[start:r.cursor]
}

func (r *CommandReader) ReadQuotedString() (string, error) {
	if !r.CanRead() {
		return "", nil
	}
	quote := r.Peek()
	if quote != '"' && quote != '\'' {
		return "", NewParsingErrorAt(tc.Translatable(mcdata.ParsingQuoteExpectedStart), r.input, r.cursor)
	}
	r.Skip()
	return r.readStringUntil(quote)
}

func (r *CommandReader) ReadString() (string, error) {
	if !r.CanRead() {
		return "", NewParsingErrorAt(tc.Translatable(mcdata.ParsingQuoteExpectedStart), r.input, r.cursor)
	}
	if ch := r.Peek(); ch == '"' || ch == '\'' {
		return r.ReadQuotedString()
	}
	return r.ReadUnquotedString(), nil
}

func (r *CommandReader) ReadWord() string {
	start := r.cursor
	for r.CanRead() && r.Peek() != ' ' {
		r.Skip()
	}
	return r.input[start:r.cursor]
}

func (r *CommandReader) PeekWord() string {
	saved := r.cursor
	word := r.ReadWord()
	r.cursor = saved
	return word
}

func (r *CommandReader) ExpectSeparator() error {
	if r.CanRead() && r.Peek() != ' ' {
		return NewParsingErrorAt(tc.Translatable(mcdata.CommandExpectedSeparator), r.input, r.cursor)
	}
	return nil
}

func (r *CommandReader) readStringUntil(terminator byte) (string, error) {
	var buf []byte
	escaped := false
	for r.CanRead() {
		ch := r.Read()
		if escaped {
			if ch != terminator && ch != '\\' {
				r.cursor--
				return "", NewParsingErrorAt(
					tc.Translatable(mcdata.ParsingQuoteEscape, tc.Text(string(ch))),
					r.input, r.cursor,
				)
			}
			buf = append(buf, ch)
			escaped = false
		} else if ch == '\\' {
			escaped = true
		} else if ch == terminator {
			return string(buf), nil
		} else {
			buf = append(buf, ch)
		}
	}
	return "", NewParsingErrorAt(tc.Translatable(mcdata.ParsingQuoteExpectedEnd), r.input, r.cursor)
}

func IsAllowedInNumericUnquotedString(c byte) bool {
	return (c >= '0' && c <= '9') || c == '-' || c == '.'
}

func IsAllowedInUnquotedString(c byte) bool {
	return IsAllowedInNumericUnquotedString(c) ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		c == '_' || c == '+'
}
