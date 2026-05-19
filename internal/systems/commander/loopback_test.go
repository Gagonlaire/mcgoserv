package commander_test

import (
	"context"
	"io"
	"testing"

	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type wordParser struct{}

func (wordParser) ID() int { return 5 }

func (wordParser) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

func (wordParser) Parse(r *CommandReader) (any, error) {
	w := r.ReadWord()
	if w == "" {
		return nil, NewParsingError(r, tc.Translatable(mcdata.ParsingQuoteExpectedStart))
	}
	return w, nil
}

type numericParser struct{}

func (numericParser) ID() int { return 3 }

func (numericParser) WriteTo(_ io.Writer) (int64, error) {
	return 0, nil
}

func (numericParser) Parse(r *CommandReader) (any, error) {
	if !r.CanRead() || r.Peek() < '0' || r.Peek() > '9' {
		return nil, NewParsingError(r, tc.Translatable(mcdata.ParsingIntExpected))
	}
	return r.ReadWord(), nil
}

func newSource() *CommandSource {
	return &CommandSource{}
}

// tp
// ├── <destination: word>     (terminal)
// └── <targets: word>
//
//	├── <destination: word> (terminal)
//	└── <location: numeric> (terminal)
func teleportTree() *Dispatcher {
	d := NewDispatcher()

	dest := Argument("destination", wordParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
		return &CommandResult{Success: 1, Result: 1}, nil // tp to entity
	})

	targets := Argument("targets", wordParser{}).Connect(
		Argument("destination2", wordParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 2}, nil // tp targets to entity
		}),
		Argument("location", numericParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 3}, nil // tp targets to coords
		}),
	)

	d.Register(Literal("tp").Connect(dest, targets))
	return d
}

func TestLoopback_TerminalCandidateWins_OnSingleArg(t *testing.T) {
	d := teleportTree()
	res, err := d.ExecuteInput(context.Background(), newSource(), "tp Alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != 1 {
		t.Fatalf("expected tp-to-entity path (Result=1), got %d", res.Result)
	}
}

// The key loopback case: at root.tp, `destination` is registered first and
// would succeed on "Alice" alone (cursor 8) — but `targets` recursed reaches
// cursor 12 by consuming "Alice 123". The deeper candidate must win even
// though it's registered second.
func TestLoopback_DeeperCandidateWins_OnTrailingArgs(t *testing.T) {
	d := teleportTree()
	res, err := d.ExecuteInput(context.Background(), newSource(), "tp Alice 123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "123" reaches both word (destination2) and numeric (location); they tie
	// at the same cursor, so registration-order tiebreaker picks destination2
	// (Result=2). The point of this test is that `targets` won over `destination`,
	// not which inner child wins.
	if res.Result != 2 {
		t.Fatalf("expected targets-path winner (Result=2), got %d", res.Result)
	}
}

func TestLoopback_NestedSamParser_DeeperWins(t *testing.T) {
	d := teleportTree()
	// "Alice Bob" — targets+destination2 path, both word parsers.
	res, err := d.ExecuteInput(context.Background(), newSource(), "tp Alice Bob")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != 2 {
		t.Fatalf("expected targets+destination2 path (Result=2), got %d", res.Result)
	}
}

func TestLoopback_AllChildrenFail_SurfacesDeepestError(t *testing.T) {
	d := NewDispatcher()
	// Both children require numeric. Trailing junk fails the second numeric.
	d.Register(Literal("cmd").Connect(
		Argument("a", numericParser{}).Connect(
			Argument("b", numericParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
				return &CommandResult{Success: 1}, nil
			}),
		),
		Argument("c", numericParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1}, nil
		}),
	))

	_, err := d.ExecuteInput(context.Background(), newSource(), "cmd 1 bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	pe, ok := err.(*CommandParsingError)
	if !ok {
		t.Fatalf("expected *CommandParsingError, got %T", err)
	}
	// "cmd 1 bad" — deepest error is the second numeric parser failing on "bad" at cursor 6.
	// Direct-c-path fails at cursor 4 ("1" parses ok but trailing " bad" makes the whole
	// subtree never reach an executable; the recursion adds an "unknown command" at cursor>=4).
	if pe.Cursor() < 6 {
		t.Fatalf("expected deepest cursor >= 6, got %d", pe.Cursor())
	}
}

// Regression: literal children take priority over argument children.
func TestLoopback_LiteralPriorityOverArguments(t *testing.T) {
	d := NewDispatcher()
	d.Register(Literal("cmd").Connect(
		Literal("alice").Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 100}, nil
		}),
		Argument("name", wordParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 200}, nil
		}),
	))

	res, err := d.ExecuteInput(context.Background(), newSource(), "cmd alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != 100 {
		t.Fatalf("expected literal path (Result=100), got %d", res.Result)
	}
}

// Mixed-parser siblings should disambiguate via fail-fast — no loopback
// needed because numericParser refuses non-digit input.
func TestLoopback_MixedParsers_FailFast(t *testing.T) {
	d := NewDispatcher()
	d.Register(Literal("set").Connect(
		Argument("count", numericParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 10}, nil
		}),
		Argument("name", wordParser{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 20}, nil
		}),
	))

	res, err := d.ExecuteInput(context.Background(), newSource(), "set 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != 10 {
		t.Fatalf("expected numeric path, got Result=%d", res.Result)
	}

	res, err = d.ExecuteInput(context.Background(), newSource(), "set foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Result != 20 {
		t.Fatalf("expected word path, got Result=%d", res.Result)
	}
}
