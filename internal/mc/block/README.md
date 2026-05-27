# Block

Conventions for the `block` package. Every block behavior registration lives here.

- https://minecraft.wiki/w/Block
- https://minecraft.wiki/w/Java_Edition_protocol/Block_actions

## File layout

- **One file per behavior family.** `behavior.go` holds `DefaultBlock`. Future
  families land as `door.go`, `log.go`, `slab.go`, …
- **Family grouping** is allowed when behaviors are closely related (e.g. all
  log variants in `log.go`, all door variants in `door.go`).
- **A unique block is a family of one.** A block that needs custom behavior
  but shares it with nothing else gets its own file (e.g. `beacon.go` with
  `BeaconBlock`). Do not bucket unrelated one-offs into a `misc.go`. A block
  that needs **no** overrides gets no file at all. `registerDefaultBlocks()`
  backfills it.
- **`block.go`** holds the `ID` type and its accessors over generated data.
  Never edit `block_gen.go` by hand.
- **`registry.go`** owns the `Behavior` table (`Register`, `Lookup`).
- **`bootstrap.go`** wires every `register*` into `RegisterAll`. Family
  registrations come first in alphabetical order. `registerDefaultBlocks()`
  always runs **last** so it backfills any ID no family claimed.

## Registration shape

```go
func registerDoors() {
    for _, name := range []string{
        "minecraft:oak_door",
        "minecraft:spruce_door",
        // …other door variants
    } {
        id, ok := FromString(name)
        if !ok {
            panic("registerDoors: unknown block " + name)
        }
        Register(id, &DoorBlock{DefaultBlock: NewDefaultBlock(id)})
    }
}
```

Explicit. No `init()` side effects. `RegisterAll` is the single entry point.

### Collision resolution

`Register(id, behavior)` **panics** on duplicate ID. Two families racing for the
same block is a programmer error, caught at boot.

To keep `DefaultBlock` from claiming every ID, `registerDefaultBlocks()`
skips IDs already in the registry:

```go
func registerDefaultBlocks() {
    for i := range registry {
        if registry[i].Name == "" {
            continue
        }
        id := ID(i)
        if _, ok := Lookup(id); ok {
            continue // already claimed by a specific family
        }
        Register(id, NewDefaultBlock(id))
    }
}
```

Bootstrap order is the rule: **families first (alphabetical), defaults last.**

## Embedding pattern

Every family type embeds `*DefaultBlock` and overrides only the hooks it
actually needs. `DefaultBlock` satisfies the full `Behavior` interface with
no-op or trivial implementations, so embedding gives you the full surface for
free.

```go
// door.go
type DoorBlock struct {
    *block.DefaultBlock
}

func (d *DoorBlock) OnPlace(ctx world.PlaceContext) world.PlaceResult {
    // door-specific placement: upper half, hinge side, facing, etc.
    // return one PlaceResult with N Writes.
}

// OnBreak, OnInteract, … inherited from *DefaultBlock.
```

Never embed by value. The `Behavior` table stores `Behavior` interface values
and the methods are pointer-receivers.

## Functional-core behavior API

Behaviors are **pure**. They take a `world.*Context`, decide what should
change, and return a result describing the effects. They never mutate world
state directly. `Dimension.PlaceBlock` / `Dimension.BreakBlock` apply the
writes.

### Contexts (defined in `internal/world/block_action.go`)

`world.PlaceContext`:

| Field        | Type            | Notes                                                       |
|--------------|-----------------|-------------------------------------------------------------|
| `Pos`        | `BlockPos`      | Target cell after face offset                               |
| `ClickedPos` | `BlockPos`      | Originally clicked cell. Invariant: `Pos = ClickedPos + Face.Vector()` |
| `Face`       | `Direction`     | Face of the existing block that was hit                     |
| `Hit`        | `[3]float32`    | Cursor position within the face                             |
| `Player`     | `*entity.Player`| Placer                                                      |
| `Hand`       | `entity.Hand`   | Main / off hand                                             |
| `UsedItem`   | `item.Stack`    | Item in the placing hand                                    |
| `View`       | `BlockView`     | Read-only state lookup (set by Dimension)                   |

#### Writing to the clicked cell instead of `Pos`

Most behaviors emit writes at `ctx.Pos`. Some need to write into the originally
clicked cell: slabs merging into a double, future replace-in-place blocks.
Read `ctx.ClickedPos`, decide, emit writes there.

When a behavior's writes do **not** include `ctx.Pos`, the server auto-broadcasts
a corrective `BlockUpdate` at `ctx.Pos` to roll back the client's prediction. No
need to emit a no-op write to "fix" the predicted cell. The place sound fires
at the first write (or at the write that lands on `ctx.Pos`, when present).

`world.BreakContext`: `{Pos, State, Breaker, Tool, View}`.

`world.InteractContext`, `world.NeighborContext`, `world.TickContext` follow
the same convention. See `block_action.go` for the exact fields.

### Results

`world.PlaceResult`:

```go
type PlaceResult struct {
    OK     bool
    Writes []BlockChange
}
```

- `OK: true` with one or more `Writes` to apply.
- `OK: false` to cancel placement. The server emits a rollback `BlockUpdate`
  at `ctx.Pos` so the client snaps the predicted block back to truth.

`world.BreakResult`:

```go
type BreakResult struct {
    OK         bool
    Changes    []BlockChange
    BERemovals []BlockPos
}
```

`world.BlockChange`:

```go
type BlockChange struct {
    Pos      BlockPos
    OldState int32  // filled by Dimension.PlaceBlock/BreakBlock
    NewState int32  // set by the behavior
}
```

Behaviors set `NewState`. The dimension reads `OldState` from the chunk
**before** applying the write, so downstream consumers (break-sound from
old state, light updates, etc.) have both sides.

### Result is additive

Results grow by adding fields. `BreakResult` already added `BERemovals`
alongside `Changes` without breaking existing call sites. The same pattern
applies to any future effect kind. Do not introduce a parallel result type
when a new field will do.

## Hooks reference

| Hook              | Context           | Result          | Status         | Fires when                                            |
|-------------------|-------------------|-----------------|----------------|-------------------------------------------------------|
| `OnPlace`         | `PlaceContext`    | `PlaceResult`   | wired          | Player uses an item on a face that resolves to a placement |
| `OnBreak`         | `BreakContext`    | `BreakResult`   | wired          | Player finishes digging (creative: single packet)     |
| `OnInteract`      | `InteractContext` | `InteractResult`| interface-only | Player right-clicks an existing block (door, lever, …)|
| `OnNeighborChange`| `NeighborContext` | `PlaceResult`   | interface-only | A neighbor's state changed (water flow, redstone, …)  |
| `OnRandomTick`    | `TickContext`     | `PlaceResult`   | interface-only | Random tick picks this block (crop growth, ice melt, …)|
| `OnScheduledTick` | `TickContext`     | `PlaceResult`   | interface-only | A previously-scheduled tick is due                    |

**Status.** `wired` hooks are dispatched by `Dimension.PlaceBlock` /
`Dimension.BreakBlock` today. `interface-only` hooks exist on the
`Behavior` interface and `DefaultBlock` supplies no-op defaults so families
can override them, but **no orchestrator dispatches them yet**. Overriding
one today compiles and tests pass, but the hook will not fire until the
corresponding orchestrator lands.

**Two shape callouts:**

1. `InteractResult` is an enum (`InteractPass` / `InteractSuccess` /
   `InteractConsume`), not a `{OK, Writes}` struct. It mirrors Mojang's
   convention: the return value drives swing animation and item cooldown,
   not world writes. An interact that needs to write blocks does so via a
   separate placement, not by stuffing writes into the interact result.

2. `OnNeighborChange`, `OnRandomTick`, `OnScheduledTick` all return
   `PlaceResult`. The name is legacy. The shape (`OK + Writes`) is the
   generic "this hook produced N block writes" result. Treat `PlaceResult`
   as the canonical write-effect result for any hook that mutates blocks.

## Adding a new family

Copy-paste checklist:

1. **Create the family file.** `internal/mc/block/<family>.go`.
2. **Define the family type, embed `*DefaultBlock`.**
   ```go
   type DoorBlock struct {
       *DefaultBlock
   }
   ```
3. **Override only the hooks the family needs.** Each override returns a
   `*Result`. Never mutate world state from a hook.
4. **Add `register<Family>()`** in the same file. Resolve each ID via
   `FromString`, then call
   `Register(id, &Family{DefaultBlock: NewDefaultBlock(id)})`.
5. **Wire it in `bootstrap.go`.** Insert `register<Family>()` into
   `RegisterAll` in alphabetical order, **before** `registerDefaultBlocks()`.
6. **Add a family test** (`<family>_test.go`) asserting result shape for
   non-trivial hooks: `OK`, `len(Writes)`, target positions, expected
   `NewState`. Behaviors are pure. Tests call methods directly, no
   `Dimension` needed.

## What goes where: `mc/block/` vs `world/`

| Lives in `mc/block/`              | Lives in `world/`                       |
|-----------------------------------|-----------------------------------------|
| `Behavior` interface              | `BlockBehavior` interface (Dimension's view) |
| `DefaultBlock` + family types     | `PlaceContext`, `BreakContext`, …       |
| `Register` / `Lookup` / registry  | `PlaceResult`, `BreakResult`, `BlockChange` |
| `ID` + generated block data       | `Dimension.PlaceBlock` / `BreakBlock`   |
| `bootstrap.go` / `RegisterAll`    | State storage (chunks, sections)        |

Behavior is **never stored in the chunk**. The chunk holds state IDs.
Behavior is resolved on demand via `block.Lookup(id)`.
