package parsers

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func TestReadOptionValue(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		want      string
		wantAfter byte // expected byte under cursor after read; 0 = EOF
	}{
		{
			name:      "simple_unquoted_stops_at_comma",
			input:     "creeper,foo=bar",
			want:      "creeper",
			wantAfter: ',',
		},
		{
			name:      "simple_unquoted_stops_at_closing_bracket",
			input:     "creeper]",
			want:      "creeper",
			wantAfter: ']',
		},
		{
			name:      "compound_value_with_nested_braces",
			input:     "{Item:{id:slime}}]",
			want:      "{Item:{id:slime}}",
			wantAfter: ']',
		},
		{
			name:      "double_quoted_string_with_embedded_close_brace",
			input:     `{Name:"a}b"}]`,
			want:      `{Name:"a}b"}`,
			wantAfter: ']',
		},
		{
			name:      "single_quoted_string_with_embedded_comma",
			input:     `{Name:'a,b'},next=x`,
			want:      `{Name:'a,b'}`,
			wantAfter: ',',
		},
		{
			name:      "backslash_escapes_quote_inside_string",
			input:     `{Name:"a\"b}c"}]`,
			want:      `{Name:"a\"b}c"}`,
			wantAfter: ']',
		},
		{
			name:      "top_level_quoted_value_short_circuits",
			input:     `"hello,world",next=x`,
			want:      "hello,world",
			wantAfter: ',',
		},
		{
			name:      "list_with_quoted_strings_containing_brackets",
			input:     `{Tags:["a]","b"]},x=1`,
			want:      `{Tags:["a]","b"]}`,
			wantAfter: ',',
		},
		{
			name:      "unterminated_string_consumes_to_eof",
			input:     `{Name:"oops`,
			want:      `{Name:"oops`,
			wantAfter: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := commander.NewCommandReader(tc.input)
			got := readOptionValue(r)
			if got != tc.want {
				t.Fatalf("value: want %q got %q", tc.want, got)
			}
			if tc.wantAfter == 0 {
				if r.CanRead() {
					t.Fatalf("want EOF, got byte %q remaining", r.Peek())
				}
				return
			}
			if !r.CanRead() {
				t.Fatalf("want byte %q under cursor, got EOF", tc.wantAfter)
			}
			if r.Peek() != tc.wantAfter {
				t.Fatalf("cursor: want byte %q got %q", tc.wantAfter, r.Peek())
			}
		})
	}
}
