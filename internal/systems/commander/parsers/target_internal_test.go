package parsers

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func parseSelectorOrFail(t *testing.T, input string) *mc.Selector {
	t.Helper()
	r := commander.NewCommandReader(input)
	sel, err := parseSelector(r)
	if err != nil {
		t.Fatalf("parseSelector(%q): %v", input, err)
	}
	return sel
}

func parseSelectorErr(t *testing.T, input string) error {
	t.Helper()
	r := commander.NewCommandReader(input)
	_, err := parseSelector(r)
	if err == nil {
		t.Fatalf("parseSelector(%q): expected error, got nil", input)
	}
	return err
}

func TestParseSelectorType(t *testing.T) {
	t.Run("positive_bare_name", func(t *testing.T) {
		sel := parseSelectorOrFail(t, "@e[type=zombie]")
		if !sel.TypeInclude.Present || sel.TypeInclude.Value != "zombie" {
			t.Fatalf("want TypeInclude=zombie, got %+v", sel.TypeInclude)
		}
		if len(sel.TypeExclude) != 0 {
			t.Fatalf("want no excludes, got %v", sel.TypeExclude)
		}
	})

	t.Run("positive_strips_minecraft_namespace", func(t *testing.T) {
		sel := parseSelectorOrFail(t, "@e[type=minecraft:zombie]")
		if !sel.TypeInclude.Present || sel.TypeInclude.Value != "zombie" {
			t.Fatalf("want TypeInclude=zombie, got %+v", sel.TypeInclude)
		}
	})

	t.Run("foreign_namespace_rejected", func(t *testing.T) {
		parseSelectorErr(t, "@e[type=mymod:foo]")
	})

	t.Run("unknown_entity_rejected", func(t *testing.T) {
		parseSelectorErr(t, "@e[type=not_an_entity_xyz]")
	})

	t.Run("negation_single", func(t *testing.T) {
		sel := parseSelectorOrFail(t, "@e[type=!zombie]")
		if sel.TypeInclude.Present {
			t.Fatalf("want no positive, got %+v", sel.TypeInclude)
		}
		if len(sel.TypeExclude) != 1 || sel.TypeExclude[0] != "zombie" {
			t.Fatalf("want excludes=[zombie], got %v", sel.TypeExclude)
		}
	})

	t.Run("negation_multiple_allowed", func(t *testing.T) {
		sel := parseSelectorOrFail(t, "@e[type=!zombie,type=!creeper]")
		if len(sel.TypeExclude) != 2 {
			t.Fatalf("want 2 excludes, got %v", sel.TypeExclude)
		}
	})

	t.Run("duplicate_positive_rejected", func(t *testing.T) {
		parseSelectorErr(t, "@e[type=zombie,type=creeper]")
	})

	t.Run("positive_after_negative_rejected", func(t *testing.T) {
		parseSelectorErr(t, "@e[type=!creeper,type=zombie]")
	})

	t.Run("negative_after_positive_rejected", func(t *testing.T) {
		parseSelectorErr(t, "@e[type=zombie,type=!creeper]")
	})

	t.Run("rejected_on_all_players", func(t *testing.T) {
		parseSelectorErr(t, "@a[type=zombie]")
	})

	t.Run("rejected_on_nearest_player", func(t *testing.T) {
		parseSelectorErr(t, "@p[type=zombie]")
	})

	t.Run("rejected_on_random_player", func(t *testing.T) {
		parseSelectorErr(t, "@r[type=zombie]")
	})

	t.Run("allowed_on_self", func(t *testing.T) {
		parseSelectorOrFail(t, "@s[type=zombie]")
	})

	t.Run("allowed_on_nearest_entity", func(t *testing.T) {
		parseSelectorOrFail(t, "@n[type=zombie]")
	})
}

func TestParseSelectorNbt(t *testing.T) {
	t.Run("positive_compound_appended", func(t *testing.T) {
		sel := parseSelectorOrFail(t, `@e[nbt={OnGround:true}]`)
		if len(sel.NbtIncludes) != 1 {
			t.Fatalf("want 1 include, got %d", len(sel.NbtIncludes))
		}
		m, ok := sel.NbtIncludes[0].(map[string]any)
		if !ok {
			t.Fatalf("want map[string]any, got %T", sel.NbtIncludes[0])
		}
		if v, ok := m["OnGround"]; !ok || v != int8(1) {
			t.Fatalf("want OnGround=int8(1), got %#v", m["OnGround"])
		}
		if len(sel.NbtExcludes) != 0 {
			t.Fatalf("want no excludes, got %v", sel.NbtExcludes)
		}
	})

	t.Run("negation_appended_to_excludes", func(t *testing.T) {
		sel := parseSelectorOrFail(t, `@e[nbt=!{OnGround:true}]`)
		if len(sel.NbtIncludes) != 0 {
			t.Fatalf("want no includes, got %v", sel.NbtIncludes)
		}
		if len(sel.NbtExcludes) != 1 {
			t.Fatalf("want 1 exclude, got %d", len(sel.NbtExcludes))
		}
	})

	t.Run("multiple_positives_allowed", func(t *testing.T) {
		sel := parseSelectorOrFail(t, `@e[nbt={A:1b},nbt={B:2b}]`)
		if len(sel.NbtIncludes) != 2 {
			t.Fatalf("want 2 includes, got %d", len(sel.NbtIncludes))
		}
	})

	t.Run("mixed_positive_and_negative_allowed", func(t *testing.T) {
		sel := parseSelectorOrFail(t, `@e[nbt={A:1b},nbt=!{B:2b}]`)
		if len(sel.NbtIncludes) != 1 || len(sel.NbtExcludes) != 1 {
			t.Fatalf("want 1 include + 1 exclude, got %d/%d",
				len(sel.NbtIncludes), len(sel.NbtExcludes))
		}
	})

	t.Run("non_compound_scalar_rejected", func(t *testing.T) {
		parseSelectorErr(t, `@e[nbt=42]`)
	})

	t.Run("non_compound_list_rejected", func(t *testing.T) {
		parseSelectorErr(t, `@e[nbt=[1,2,3]]`)
	})

	t.Run("malformed_snbt_rejected", func(t *testing.T) {
		parseSelectorErr(t, `@e[nbt={unterminated]`)
	})

	t.Run("compound_with_quoted_string_containing_brace", func(t *testing.T) {
		sel := parseSelectorOrFail(t, `@e[nbt={Name:"a}b"}]`)
		if len(sel.NbtIncludes) != 1 {
			t.Fatalf("want 1 include, got %d", len(sel.NbtIncludes))
		}
		m := sel.NbtIncludes[0].(map[string]any)
		if v, _ := m["Name"].(string); v != "a}b" {
			t.Fatalf("want Name=a}b, got %#v", m["Name"])
		}
	})
}

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
