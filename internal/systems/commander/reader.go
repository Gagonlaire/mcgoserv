package commander

import (
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/text-component"
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
	for r.CanRead() && isAllowedInUnquotedString(r.Peek()) {
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
		return "", r.makeError("expected a quote to start a string")
	}
	r.Skip()
	return r.readStringUntil(quote)
}

func (r *CommandReader) ReadString() (string, error) {
	if !r.CanRead() {
		return "", r.makeError("expected a string")
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

func (r *CommandReader) Expect(expected byte) error {
	if !r.CanRead() {
		return r.makeError("expected '" + string(expected) + "'")
	}
	if r.Peek() != expected {
		return r.makeError("expected '" + string(expected) + "' but got '" + string(r.Peek()) + "'")
	}
	r.Skip()
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
				return "", r.makeError("invalid escape sequence '\\" + string(ch) + "'")
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
	return "", r.makeError("unclosed quoted string")
}

func (r *CommandReader) makeError(message string) *CommandParseError {
	return NewParseError(tc.Text(message), r.input, r.cursor)
}

func isAllowedInUnquotedString(c byte) bool {
	return (c >= '0' && c <= '9') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		c == '_' || c == '-' || c == '.' || c == '+'
}
