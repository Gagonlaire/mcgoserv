package commander_test

import (
	"context"
	"io"
	"strings"
	"testing"

	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func mustPanic(t *testing.T, contains string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q, got none", contains)
		}
		msg, ok := r.(error)
		var s string
		if ok {
			s = msg.Error()
		} else {
			s = string(asString(r))
		}
		if !strings.Contains(s, contains) {
			t.Fatalf("expected panic containing %q, got %q", contains, s)
		}
	}()
	fn()
}

func asString(v any) []byte {
	switch x := v.(type) {
	case string:
		return []byte(x)
	case error:
		return []byte(x.Error())
	}
	return []byte{}
}

func TestBuild_SimpleRequired(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/kill <target>", wordParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: int(cc.Args["target"].(string)[0])}, nil
		})
	})

	res, err := d.ExecuteInput(context.Background(), newSource(), "kill Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success != 1 || res.Result != int('A') {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestBuild_OptionalLadder_SingleFn(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/effect give <entity> [<seconds>] [<amp>]", wordParser{}, numericParser{}, numericParser{}).
			Executes(func(cc *CommandContext) (*CommandResult, error) {
				secs := GetArgumentOr[string](cc.Args, "seconds", "")
				amp := GetArgumentOr[string](cc.Args, "amp", "")
				code := 0
				if secs != "" {
					code |= 1
				}
				if amp != "" {
					code |= 2
				}
				return &CommandResult{Success: 1, Result: code}, nil
			})
	})

	cases := []struct {
		input  string
		expect int
	}{
		{"effect give Alice", 0},
		{"effect give Alice 10", 1},
		{"effect give Alice 10 2", 3},
	}
	for _, c := range cases {
		res, err := d.ExecuteInput(context.Background(), newSource(), c.input)
		if err != nil {
			t.Fatalf("%q: unexpected error: %v", c.input, err)
		}
		if res.Result != c.expect {
			t.Fatalf("%q: expected Result=%d, got %d", c.input, c.expect, res.Result)
		}
	}
}

func TestBuild_NamedChoice_BindsArg(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/difficulty <level: peaceful|easy|normal|hard>").
			Executes(func(cc *CommandContext) (*CommandResult, error) {
				lvl := cc.Args["level"].(string)
				code := map[string]int{"peaceful": 0, "easy": 1, "normal": 2, "hard": 3}[lvl]
				return &CommandResult{Success: 1, Result: code}, nil
			})
	})

	cases := map[string]int{"difficulty peaceful": 0, "difficulty easy": 1, "difficulty hard": 3}
	for input, want := range cases {
		res, err := d.ExecuteInput(context.Background(), newSource(), input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if res.Result != want {
			t.Fatalf("%q: expected %d, got %d", input, want, res.Result)
		}
	}
}

func TestBuild_AnonymousChoice_ExecutesEach(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/gamemode <survival|creative|adventure>").ExecutesEach(
			func(*CommandContext) (*CommandResult, error) { return &CommandResult{Success: 1, Result: 0}, nil },
			func(*CommandContext) (*CommandResult, error) { return &CommandResult{Success: 1, Result: 1}, nil },
			func(*CommandContext) (*CommandResult, error) { return &CommandResult{Success: 1, Result: 2}, nil },
		)
	})

	cases := map[string]int{"gamemode survival": 0, "gamemode creative": 1, "gamemode adventure": 2}
	for input, want := range cases {
		res, err := d.ExecuteInput(context.Background(), newSource(), input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if res.Result != want {
			t.Fatalf("%q: expected %d, got %d", input, want, res.Result)
		}
	}
}

func TestBuild_PrefixMerge_TwoCalls(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/data get").Executes(func(*CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 10}, nil
		})
		Build("/data merge <nbt>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 20}, nil
		})
	})

	res, _ := d.ExecuteInput(context.Background(), newSource(), "data get")
	if res.Result != 10 {
		t.Fatalf("expected 10, got %d", res.Result)
	}
	res, _ = d.ExecuteInput(context.Background(), newSource(), "data merge x")
	if res.Result != 20 {
		t.Fatalf("expected 20, got %d", res.Result)
	}
}

func TestBuild_Variadic(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/say <msg ...>", greedyParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: len(cc.Args["msg"].(string))}, nil
		})
	})

	res, err := d.ExecuteInput(context.Background(), newSource(), "say hello world this is a test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != len("hello world this is a test") {
		t.Fatalf("expected %d, got %d", len("hello world this is a test"), res.Result)
	}
}

func TestBuild_Panic_ParserCountMismatch(t *testing.T) {
	mustPanic(t, "expected 2 parsers, got 1", func() {
		Build("/foo <x> <y>", wordParser{})
	})
}

func TestBuild_Panic_RequiredAfterOptional(t *testing.T) {
	mustPanic(t, "appears after an optional", func() {
		Build("/foo [<x>] <y>", wordParser{}, wordParser{})
	})
}

func TestBuild_Panic_OptionalMissingAngleBrackets(t *testing.T) {
	mustPanic(t, "must wrap an angle placeholder", func() {
		Build("/foo [x]", wordParser{})
	})
}

func TestBuild_Panic_VariadicNotLast(t *testing.T) {
	mustPanic(t, "must be the last token", func() {
		Build("/foo <x ...> <y>", wordParser{}, wordParser{})
	})
}

func TestBuild_Panic_AnonymousChoice_RequiresExecutesEach(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "anonymous choice requires ExecutesEach", func() {
		d.RegisterBuilders(func() {
			Build("/foo <a|b>").Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
		})
	})
}

func TestBuild_Panic_DuplicateLeaf(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "already has an execute fn", func() {
		d.RegisterBuilders(func() {
			Build("/foo <x>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
			Build("/foo <x>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
		})
	})
}

func TestBuild_Panic_DifferentParserSameName(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "different parser", func() {
		d.RegisterBuilders(func() {
			Build("/foo <x>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
			Build("/foo <x> <y>", numericParser{}, wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
		})
	})
}

func TestBuild_Panic_LiteralVsArgumentCollision(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "cannot add literal", func() {
		d.RegisterBuilders(func() {
			Build("/foo <x>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
			Build("/foo bar").Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
		})
	})
}

func TestBuild_Panic_ChoiceBindKeyConflict(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "already binds to", func() {
		d.RegisterBuilders(func() {
			Build("/foo <a: x|y>").Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
			Build("/foo <b: x|z>").Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
		})
	})
}

func TestBuild_Panic_ExecutesEachCountMismatch(t *testing.T) {
	d := NewDispatcher()
	mustPanic(t, "ExecutesEach expected", func() {
		d.RegisterBuilders(func() {
			Build("/foo <a|b|c>").ExecutesEach(
				func(*CommandContext) (*CommandResult, error) { return nil, nil },
				func(*CommandContext) (*CommandResult, error) { return nil, nil },
			)
		})
	})
}

func TestBuild_Description_OverrideOnRoot(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/foo <x>", wordParser{}).Executes(func(*CommandContext) (*CommandResult, error) {
			return nil, nil
		})
		Build("/bar <y>", wordParser{}).
			Description("custom line").
			Executes(func(*CommandContext) (*CommandResult, error) {
				return nil, nil
			})
	})

	if d.Resolve("foo").Description != "" {
		t.Fatalf("foo root should have no description, got %q", d.Resolve("foo").Description)
	}
	if d.Resolve("bar").Description != "custom line" {
		t.Fatalf("bar root override wrong: %q", d.Resolve("bar").Description)
	}
}

func TestBuild_SmartUsage_CollapsesOptionalChain(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/effect <e> [<s>] [<a>]", wordParser{}, numericParser{}, numericParser{}).
			Executes(func(*CommandContext) (*CommandResult, error) { return nil, nil })
	})

	lines := d.UsageLines(d.Root, "", 0)
	if len(lines) != 1 || lines[0] != "/effect <e> [<s>]" {
		t.Fatalf("expected [/effect <e> [<s>]], got %v", lines)
	}
}

func TestBuild_SmartUsage_RequiredChain(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/give <targets> <item>", wordParser{}, wordParser{}).
			Executes(func(*CommandContext) (*CommandResult, error) { return nil, nil })
	})

	lines := d.UsageLines(d.Root, "", 0)
	if len(lines) != 1 || lines[0] != "/give <targets> <item>" {
		t.Fatalf("expected [/give <targets> <item>], got %v", lines)
	}
}

func TestBuild_SmartUsage_MultiBranchGroups(t *testing.T) {
	d := NewDispatcher()
	d.RegisterBuilders(func() {
		Build("/data get").Executes(func(*CommandContext) (*CommandResult, error) { return nil, nil })
		Build("/data merge").Executes(func(*CommandContext) (*CommandResult, error) { return nil, nil })
	})

	lines := d.UsageLines(d.Root, "", 0)
	if len(lines) != 1 || lines[0] != "/data (get|merge)" {
		t.Fatalf("expected [/data (get|merge)], got %v", lines)
	}
}

type greedyParser struct{}

func (greedyParser) ID() int { return 5 }

func (greedyParser) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (greedyParser) Parse(r *CommandReader) (any, error) {
	rem := r.GetRemaining()
	r.SetCursor(r.TotalLength())
	return rem, nil
}
