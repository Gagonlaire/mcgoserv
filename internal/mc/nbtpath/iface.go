package nbtpath

import "reflect"

// MetaEqual deep-compares two snapshot values for the diff-then-mark hook.
// Used by generated MetaDiffMark methods. Treats nil-vs-zero-value as inequality.
func MetaEqual(a, b any) bool { return reflect.DeepEqual(a, b) }

// NbtSource is the read-only NBT surface.
// NOTE: Player only implement NbtSource, writes are forbidden
type NbtSource interface {
	NbtRoot() (any, error)
	NbtGet(p Path) (any, error)
}

// NbtTarget extends NbtSource with mutation.
type NbtTarget interface {
	NbtSource
	NbtSet(p Path, v any) (int, error)
	NbtAppend(p Path, v any) (int, error)
	NbtPrepend(p Path, v any) (int, error)
	NbtInsert(p Path, idx int, v any) (int, error)
	NbtRemove(p Path) (int, error)
	NbtMerge(src map[string]any) (int, error)
	NbtMergeAt(p Path, src map[string]any) (int, error)
}
