package nbtpath_test

import (
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/mc/nbtpath"
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
)

func TestPlainSink(t *testing.T) {
	t.Run("int8_renders_with_b_suffix", func(t *testing.T) {
		got := string(nbtpath.FormatSNBT(int8(5)))
		if got != "5b" {
			t.Fatalf("want 5b got %q", got)
		}
	})

	t.Run("string_escapes_quote_and_backslash", func(t *testing.T) {
		got := string(nbtpath.FormatSNBT(`he said "hi" \\ done`))
		want := `"he said \"hi\" \\\\ done"`
		if got != want {
			t.Fatalf("want %q got %q", want, got)
		}
	})

	t.Run("compound_keys_sorted_alphabetically", func(t *testing.T) {
		// Run many times to catch nondeterministic iteration.
		for i := 0; i < 50; i++ {
			got := string(nbtpath.FormatSNBT(map[string]any{
				"b": int8(1), "a": int8(2), "c": int8(3),
			}))
			want := "{a: 2b, b: 1b, c: 3b}"
			if got != want {
				t.Fatalf("iter %d: want %q got %q", i, want, got)
			}
		}
	})

	t.Run("byte_array_uses_B_prefix", func(t *testing.T) {
		got := string(nbtpath.FormatSNBT([]byte{1, 2, 3}))
		want := "[B; 1b, 2b, 3b]"
		if got != want {
			t.Fatalf("want %q got %q", want, got)
		}
	})

	t.Run("int_array_uppercase_header_no_value_suffix", func(t *testing.T) {
		got := string(nbtpath.FormatSNBT([]int32{1, 1, 1, 1}))
		want := "[I; 1, 1, 1, 1]"
		if got != want {
			t.Fatalf("want %q got %q", want, got)
		}
	})

	t.Run("list_of_compounds_round_trips_through_SNBTToValue", func(t *testing.T) {
		input := []any{
			map[string]any{"id": "a", "Slot": int8(0)},
			map[string]any{"id": "b", "Slot": int8(1)},
		}
		snbt := nbtpath.FormatSNBT(input)
		got, err := nbtpath.SNBTToValue(snbt)
		if err != nil {
			t.Fatalf("SNBTToValue: %v", err)
		}
		gotList, ok := got.([]any)
		if !ok {
			t.Fatalf("want []any got %T", got)
		}
		if len(gotList) != 2 {
			t.Fatalf("want 2 elems got %d", len(gotList))
		}
		first, _ := gotList[0].(map[string]any)
		if first["id"] != "a" || first["Slot"] != int8(0) {
			t.Fatalf("first elem mismatch: %#v", first)
		}
	})
}

func extras(c *tc.TextComponent) []*tc.TextComponent {
	out := make([]*tc.TextComponent, len(c.Extra))
	for i, e := range c.Extra {
		out[i] = e.(*tc.TextComponent)
	}
	return out
}

func TestComponentSink(t *testing.T) {
	t.Run("number_emits_color_gold_value_then_color_red_suffix", func(t *testing.T) {
		got := nbtpath.FormatSNBTComponent(int8(5))
		ex := extras(got)
		if len(ex) != 2 {
			t.Fatalf("want 2 extras got %d", len(ex))
		}
		if ex[0].Text != "5" || ex[0].Color != tc.ColorGold {
			t.Fatalf("first extra: want gold 5 got %q color=%q", ex[0].Text, ex[0].Color)
		}
		if ex[1].Text != "b" || ex[1].Color != tc.ColorRed {
			t.Fatalf("second extra: want red b got %q color=%q", ex[1].Text, ex[1].Color)
		}
	})

	t.Run("safe_key_emits_unquoted_aqua", func(t *testing.T) {
		got := nbtpath.FormatSNBTComponent(map[string]any{"foo": int8(5)})
		found := false
		for _, e := range extras(got) {
			if e.Text == "foo" && e.Color == tc.ColorAqua {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected aqua 'foo' key in extras, got %#v", extras(got))
		}
	})

	t.Run("key_with_space_emits_quoted_with_punct_quotes", func(t *testing.T) {
		got := nbtpath.FormatSNBTComponent(map[string]any{"a b": int8(5)})
		// Should NOT contain an aqua key. The key must be rendered as a green
		// string surrounded by white quote punctuation.
		var greenStr *tc.TextComponent
		for _, e := range extras(got) {
			if e.Color == tc.ColorAqua {
				t.Fatalf("unexpected aqua extra %q (key should be quoted)", e.Text)
			}
			if e.Text == "a b" && e.Color == tc.ColorGreen {
				greenStr = e
			}
		}
		if greenStr == nil {
			t.Fatalf("expected green 'a b' string-segment in extras")
		}
	})
}

func TestSNBTCodec(t *testing.T) {
	t.Run("snbt_to_value_int_returns_int32", func(t *testing.T) {
		v, err := nbtpath.SNBTToValue("42")
		if err != nil {
			t.Fatalf("SNBTToValue: %v", err)
		}
		if v != int32(42) {
			t.Fatalf("want int32(42) got %T(%v)", v, v)
		}
	})

	t.Run("snbt_to_value_invalid_returns_error", func(t *testing.T) {
		_, err := nbtpath.SNBTToValue("")
		if err == nil {
			t.Fatal("want error got nil")
		}
	})
}
