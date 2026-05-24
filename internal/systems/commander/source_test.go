package commander_test

import (
	"context"
	"io"
	"testing"

	"github.com/Gagonlaire/mcgoserv/internal/proto"
	. "github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

func TestCommandSource_WithBuildersAreIndependent(t *testing.T) {
	base := &CommandSource{
		Position:        [3]float64{1, 2, 3},
		Rotation:        [2]float32{10, 20},
		Anchor:          AnchorFeet,
		PermissionLevel: 4,
	}

	mod := base.WithPosition([3]float64{99, 99, 99}).WithAnchor(AnchorEyes).WithDimension("minecraft:the_nether")

	if base.Position != ([3]float64{1, 2, 3}) {
		t.Fatalf("base position mutated: %v", base.Position)
	}
	if base.Anchor != AnchorFeet {
		t.Fatalf("base anchor mutated: %v", base.Anchor)
	}
	if base.Dimension != "" {
		t.Fatalf("base dimension mutated: %q", base.Dimension)
	}
	if mod.Position != ([3]float64{99, 99, 99}) {
		t.Fatalf("modified position wrong: %v", mod.Position)
	}
	if mod.Anchor != AnchorEyes {
		t.Fatalf("modified anchor wrong: %v", mod.Anchor)
	}
	if mod.Dimension != proto.Identifier("minecraft:the_nether") {
		t.Fatalf("modified dimension wrong: %q", mod.Dimension)
	}
	// Rotation is unchanged on mod — should still carry the base value.
	if mod.Rotation != ([2]float32{10, 20}) {
		t.Fatalf("modified lost unchanged rotation: %v", mod.Rotation)
	}
}

type recordingConsumer struct {
	calls []consumerCall
}

type consumerCall struct {
	src     *CommandSource
	success bool
	result  int
}

func (c *recordingConsumer) OnResult(src *CommandSource, success bool, result int) {
	c.calls = append(c.calls, consumerCall{src, success, result})
}

type fixedSrc struct{}

func (fixedSrc) ID() int { return 5 }

func (fixedSrc) WriteTo(_ io.Writer) (int64, error) { return 0, nil }

func (fixedSrc) Parse(r *CommandReader) (any, error) { return r.ReadWord(), nil }

func TestExecute_ResultConsumerFiresPerBranch(t *testing.T) {
	consumer := &recordingConsumer{}
	root := &CommandSource{PermissionLevel: 4, ResultConsumer: consumer}

	d := NewDispatcher()
	// `cmd <name>` returns Result=7. Modifier multiplies source ×3 — three branches.
	forkModifier := RedirectModifier(func(_ context.Context, src *CommandSource) ([]*CommandSource, error) {
		return []*CommandSource{src, src, src}, nil
	})
	leaf := Literal("leaf").Connect(
		Argument("name", fixedSrc{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			return &CommandResult{Success: 1, Result: 7}, nil
		}),
	)
	d.Register(Literal("cmd").Modify(forkModifier).Fork().Redirects(leaf))

	res, err := d.ExecuteInput(context.Background(), root, "cmd hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Success != 3 {
		t.Fatalf("expected Success=3 (3 branches), got %d", res.Success)
	}
	if len(consumer.calls) != 3 {
		t.Fatalf("expected 3 consumer calls, got %d", len(consumer.calls))
	}
	for i, c := range consumer.calls {
		if !c.success {
			t.Fatalf("call %d: expected success=true, got false", i)
		}
		if c.result != 7 {
			t.Fatalf("call %d: expected result=7, got %d", i, c.result)
		}
	}
}

func TestExecute_ModifierAppliesWithoutFork(t *testing.T) {
	var seenPos [3]float64

	transformed := [3]float64{42, 42, 42}
	posModifier := RedirectModifier(func(_ context.Context, src *CommandSource) ([]*CommandSource, error) {
		return []*CommandSource{src.WithPosition(transformed)}, nil
	})

	d := NewDispatcher()
	leaf := Literal("leaf").Connect(
		Argument("name", fixedSrc{}).Executes(func(cc *CommandContext) (*CommandResult, error) {
			seenPos = cc.Source.Position
			return &CommandResult{Success: 1, Result: 1}, nil
		}),
	)
	// NOTE: Modify but NOT Fork() — single-source transform.
	d.Register(Literal("cmd").Modify(posModifier).Redirects(leaf))

	_, err := d.ExecuteInput(context.Background(), &CommandSource{PermissionLevel: 4}, "cmd hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenPos != transformed {
		t.Fatalf("expected modifier to transform position to %v, got %v", transformed, seenPos)
	}
}
