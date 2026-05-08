# nbtpath

NBT path resolution and access for `/data` and any other system that needs to
read/write NBT values at runtime.

## Requirements

To match vanilla behavior and support the full `/data` command, this package:

- Supports the **full Mojang path grammar**: dotted member access, indexed
  list access, list-all (`[]`), compound self-match (`{filter}`), and
  list-element match (`[{filter}]`).
- Resolve **multi-match paths** correctly: `/data modify entity @e[…]
  Items[].id set value "diamond"` must visit every matching tag, and the
  number of writes must be reported back so `/execute store result` works.
- Make the **write-back path flip the right metadata dirty bits** so
  `/data merge` actually pushes a metadata packet to the client.
- Render output as **colored SNBT text components** with the same
  vanilla-style formatting players expect (lowercase value suffixes,
  `[I; …]` array headers).

## Architecture

### Path representation

Path steps are a sum type: one concrete struct per kind of step, all
satisfying `PathStep`:

| Step          | Source syntax       | Meaning                          |
|---------------|---------------------|----------------------------------|
| `MemberStep`  | `key`               | Pick a named field of a compound |
| `IndexStep`   | `[3]`               | Pick element 3 of a list         |
| `AllStep`     | `[]`                | Every element of a list          |
| `SelfMatch`   | `{filter}`          | Self if compound matches filter  |
| `MatchAll`    | `[{filter}]`        | List elements matching filter    |

Filters are decoded from SNBT to `map[string]any` *once* at parse time to increase
efficiency. The path resolver can then compare against that map directly without re-parsing SNBT at every step.

### Resolution to anchors

`Resolve(root, path)` walks the path and returns a slice of `Anchor`
values, one per match. An `Anchor` carries:

- `Parent` the containing map or list,
- `Key` set when the parent is a compound,
- `Index` set when the parent is a list (`-1` otherwise).

Mutation operations (`Set`, `Append`, `Prepend`, `Insert`, `Remove`,
`MergeAt`) iterate the anchor list and act on each one. They return
`(int, error)` — the int is the number of tags actually written or
deleted, which flows out to `/execute store result`.

This approach offers:

- **Direct write access**: the parent reference lets us mutate without
  re-walking the path.
- **No callbacks**: each operation is a plain method on the package, not
  a visitor with a closure parameter.

### Currency between layers

Inside the package, everything speaks plain Go values: `any`,
`map[string]any`, `[]any`, primitives like `int8` / `int32` / `float64`,
and the typed array slices `[]byte` / `[]int32` / `[]int64`. The
`nbt.StringifiedMessage` (raw SNBT text) only appears at two boundaries:

- the **parser**, which consumes user input,
- the **formatter**, which produces output for the player.

This is efficient because it avoids heavy parsing during process and a clean (even tho harder to understand) separation of concerns.

### The Sink-based formatter

`WriteSNBT(sink, value)` is a single walker that emits typed segments
(`Punct`, `Key`, `String`, `Number`, `Type`) to a `Sink`. Two sinks are
provided:

- `PlainSink` writes to a `strings.Builder` and produces a plain SNBT
  string (`FormatSNBT`).
- `ComponentSink` writes to a `*tc.TextComponent` with vanilla-style
  colors (`FormatSNBTComponent`):
  - keys: aqua,
  - quoted strings: green between white quote punctuation,
  - numbers: gold value followed by red lowercase type suffix
    (`5b`, `300s`, `42l`, `1.5f`, `3.14d`),
  - punctuation (`{ } [ ] : , ;`): white,
  - array headers: white `[`, red uppercase letter, white `; ` (`[B; 1b, 2b]`, `[I; 1, 1, 1, 1]`, `[L; 1l, 2l]`).

Not a vanilla spec, compound keys are sorted alphabetically before output so plain-string
formatting is deterministic, useful for tests.

### Source / Target split

Two small interfaces decide who can be read from and who can be
written to:

```go
package nbtpath

type Path struct {}

type NbtSource interface {
    NbtRoot() (any, error)
    NbtGet(p Path) (any, error)
}

type NbtTarget interface {
    NbtSource
    NbtSet(p Path, v any)        (int, error)
    NbtAppend(p Path, v any)     (int, error)
    NbtPrepend(p Path, v any)    (int, error)
    NbtInsert(p Path, idx int, v any) (int, error)
    NbtRemove(p Path)            (int, error)
    NbtMerge(src map[string]any) (int, error)
    NbtMergeAt(p Path, src map[string]any) (int, error)
}
```

This pattern allow to easily block writing to players

`ValueSource`: one of `LiteralValueSource` (parsed from `value <nbt>`),
`FromValueSource` (read from another `NbtSource` at a path), abstracts
the right-hand side of `/data modify`. Block, entity, and storage
sources all plug in uniformly.

### Errors as Go sentinels

The package returns plain Go sentinels (`ErrPathNotFound`, `ErrNotAList`,
`ErrNotACompound`, `ErrIndexOOB`, `ErrEmptyPath`, `ErrSelectsRoot`,
`ErrExpectedCompound`). The `/data` command layer maps these to the
right `mcdata` translation key for player-visible error messages.

## Example: `/data modify entity <target> Health set value 12.5f`

Starting from a player typing the command:

1. **Parser** (`internal/systems/commander/parsers/nbt.go`) reads
   `Health` and produces `Path{Steps: [MemberStep{Name:"Health"}]}`.
   It reads `12.5f` and produces an `nbt.StringifiedMessage`.
2. **Command handler** (`internal/server/commands/data.go`) resolves the
   target entity, asserts it implements `NbtTarget`, and rejects players.
3. The handler calls `nbtpath.SNBTToValue("12.5f")` once, gets
   `float32(12.5)`.
4. The handler calls `target.NbtSet(path, float32(12.5))`.
5. The generated `NbtSet` on the entity runs the snapshot → marshal →
   `Set` → write-back → diff-then-mark sequence.
6. `Set` calls `Resolve(root, path)`, gets one anchor for `Health`,
   writes the new value, returns `1`.
7. `NbtSet` returns `(1, nil)` to the handler.
8. The handler reports the `commands.data.entity.modified` translation
   to the player and stuffs `1` into `CommandResult.Result` for
   `/execute store result`.
