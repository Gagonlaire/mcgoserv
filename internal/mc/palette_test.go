package mc

import (
	"bytes"
	"testing"
)

// fillToDirect mutates pc with enough distinct values to force an upgrade into a DirectContainer.
func fillToDirect(t *testing.T, pc *PalettedContainer) []int32 {
	t.Helper()

	const distinct = 257
	values := make([]int32, distinct)
	for i := 0; i < distinct; i++ {
		v := int32(i + 1)
		values[i] = v
		if err := pc.Set(i, v); err != nil {
			t.Fatalf("Set(%d, %d): %v", i, v, err)
		}
	}
	if _, ok := pc.c.(*DirectContainer); !ok {
		t.Fatalf("expected DirectContainer after %d distinct sets, got %T", distinct, pc.c)
	}
	return values
}

func TestIndirectContainerWireOrder(t *testing.T) {
	pc := NewPalettedContainer(9) // single value: dirt
	if err := pc.Set(0, 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	var buf bytes.Buffer
	if _, err := pc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	data := buf.Bytes()

	if len(data) < 1 {
		t.Fatalf("encoded too short: %d bytes", len(data))
	}
	if got := int(data[0]); got != MinIndirectBits {
		t.Fatalf("BitsPerEntry: got %d, want %d", got, MinIndirectBits)
	}

	r := bytes.NewReader(data[1:])

	var paletteCount VarInt
	if _, err := paletteCount.ReadFrom(r); err != nil {
		t.Fatalf("read palette count: %v", err)
	}
	if paletteCount != 2 {
		t.Fatalf("palette count: got %d, want 2", paletteCount)
	}
	wantEntries := []VarInt{9, 0}
	for i, want := range wantEntries {
		var got VarInt
		if _, err := got.ReadFrom(r); err != nil {
			t.Fatalf("read palette[%d]: %v", i, err)
		}
		if got != want {
			t.Fatalf("palette[%d]: got %d, want %d", i, got, want)
		}
	}

	if r.Len() != 2048 {
		t.Fatalf("data array bytes: got %d, want 2048 (remaining after palette)", r.Len())
	}
}

func TestDirectContainerWireOrder(t *testing.T) {
	pc := NewPalettedContainer(0)
	values := fillToDirect(t, pc)

	for i, want := range values {
		if got := pc.Get(i); got != want {
			t.Fatalf("Get(%d): got %d, want %d", i, got, want)
		}
	}

	var buf bytes.Buffer
	if _, err := pc.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	data := buf.Bytes()

	if got := int(data[0]); got != DirectBits {
		t.Fatalf("BitsPerEntry: got %d, want %d", got, DirectBits)
	}
	const wantLongBytes = 1024 * 8
	if got := len(data) - 1; got != wantLongBytes {
		t.Fatalf("data array bytes: got %d, want %d", got, wantLongBytes)
	}
}

// TestPalettedContainerSizeMatchesEncoded verifies Size() returns the exact
// byte length produced by WriteTo across every container variant.
func TestPalettedContainerSizeMatchesEncoded(t *testing.T) {
	cases := []struct {
		name  string
		setup func(*testing.T) *PalettedContainer
	}{
		{
			"single",
			func(*testing.T) *PalettedContainer { return NewPalettedContainer(9) },
		},
		{
			"indirect_after_set",
			func(*testing.T) *PalettedContainer {
				pc := NewPalettedContainer(9)
				_ = pc.Set(0, 0)
				_ = pc.Set(1, 5)
				return pc
			},
		},
		{
			"indirect_after_resize",
			func(*testing.T) *PalettedContainer {
				// Force palette resize past MinIndirectBits (16 slots) so the
				// data array is rebuilt at a wider BPE before encoding.
				pc := NewPalettedContainer(0)
				for i := 0; i < 20; i++ {
					_ = pc.Set(i, int32(i+1))
				}
				return pc
			},
		},
		{
			"direct_after_upgrade",
			func(t *testing.T) *PalettedContainer {
				pc := NewPalettedContainer(0)
				fillToDirect(t, pc)
				return pc
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pc := tc.setup(t)
			var buf bytes.Buffer
			if _, err := pc.WriteTo(&buf); err != nil {
				t.Fatalf("WriteTo: %v", err)
			}
			if pc.Size() != buf.Len() {
				t.Fatalf("Size mismatch: Size()=%d, encoded=%d", pc.Size(), buf.Len())
			}
		})
	}
}
