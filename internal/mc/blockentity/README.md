# Block entity

Conventions for the `blockentity` package. Every block-entity family
registration lives here.

- https://minecraft.wiki/w/Block_entity
- https://minecraft.wiki/w/Block_entity_format

## Architecture

A **block entity** (BE) is extra state attached to a block at a position,
holds what block states cannot: inventories, sign text, banner patterns,
cooking progress.

The abstraction is split across two packages, mirroring the block split:

| Lives in `world/`                          | Lives in `mc/blockentity/`                          |
|--------------------------------------------|------------------------------------------------------|
| `BlockEntity` interface (`Pos`, `Type`)    | `Type` alias (`= world.BEType`) + named constants    |
| `BEType` named type (uint8)                | Capability interfaces (`Ticker`, `Container`, …)     |
| `Chunk.BlockEntities []BlockEntity`        | Registry (`Register`, `Lookup`, `Factory`)           |
| `PlaceResult.BEAdds` / `BreakResult.BERemovals` | Generated `type_gen.go`, family files            |
| `removeBlockEntity` / `addBlockEntity`     | `bootstrap.go` / `RegisterAll`                       |

`world` owns chunk storage and the minimal interface chunks store. `mc/blockentity`
owns the domain interfaces concrete BE types implement.

## Capability interfaces

A concrete BE type implements `world.BlockEntity` plus whichever capability
interfaces apply. Chunk-level iteration uses type assertions: concrete types
pay zero cost for capabilities they do not implement.

| Interface     | Method                                         | When implemented                          |
|---------------|------------------------------------------------|-------------------------------------------|
| `Ticker`      | `Tick(ctx world.TickContext)`                  | Furnace, hopper, beacon, conduit, spawner |
| `Container`   | `Inventory() *container.Instance`              | Chest, furnace, hopper, barrel, dropper   |
| `Interactable`| `OnInteract(ctx world.InteractContext) InteractResult` | Sign (edit text)                  |
| `Networked`   | `NetworkData() any`                            | Sign, banner, head, beacon                |
| `Persistent`  | `Marshal() any` / `Unmarshal(any) error`       | Every BE — seam, no impls yet             |

Reuse `world.TickContext` / `world.InteractContext`, same types blocks use.
The BE instance is `self` via the method receiver: the context carries world
access only.

TODO: write-back mechanism if a `Ticker` ever needs to schedule writes.
TODO: bucketed `Chunk.TickingBEs` list to skip the scan-and-assert per tick
once concrete tickers exist and density justifies it.

## Type enum

`Type` is `uint8` (aliased to `world.BEType`). Constants generated from
`internal/mcdata/reports/registries.json` (`minecraft:block_entity_type`)
by `cmd/gen-block-entity-types`. Constant value matches the vanilla
`protocol_id`, so `Type` is the wire VarInt directly — no translation at
encode time.

## Registry

```go
var factories [len(registry)]Factory

func Register(t Type, f Factory)
func Lookup(t Type) (Factory, bool)
```

Flat sparse array indexed by `Type`. `Factory` constructs a BE from a position
and an NBT root, used by the region-loader to materialise BEs from disk.

**Placement does not go through the factory.** Block `OnPlace` returns the
constructed BE via `PlaceResult.BEAdds`, `Dimension.PlaceBlock` attaches it
to the owning chunk after writes. Block behavior already knows everything
needed to construct the BE (orientation, facing, owner), so funneling through
a factory just to re-derive that is indirection.

TODO: switch to a dense table with mandatory factories once every BE type has
a concrete implementation.

## Bootstrap

`bootstrap.go` exports `RegisterAll`, called from `main.go` alongside
`block.RegisterAll`. Family registrations land here as concrete BE types
arrive. No `init()` side effects.

## Adding a new family

Mirror `internal/mc/block/README.md` "Adding a new family":

1. **Create the family file.** `internal/mc/blockentity/<family>.go`.
2. **Define the family type.** Embed nothing, BEs hold their own state.
   ```go
   type Chest struct {
       pos world.BlockPos
       inv container.Instance
   }
   func (c *Chest) Pos() world.BlockPos       { return c.pos }
   func (c *Chest) Type() Type                { return TypeChest }
   func (c *Chest) Inventory() *container.Instance { return &c.inv }
   ```
3. **Implement only the capability interfaces the family needs.** A chest is a
   `Container`. A furnace is a `Container` + `Ticker`. A sign is `Networked`
   + `Interactable`.
4. **Add `register<Family>()`** in the same file.
   ```go
   func registerChest() {
       Register(TypeChest, func(pos world.BlockPos, _ any) (world.BlockEntity, error) {
           return &Chest{pos: pos, inv: container.NewInstance(27)}, nil
       })
   }
   ```
5. **Wire into `bootstrap.go`** in alphabetical order.
6. **Block side** (e.g. `mc/block/chest.go`): `ChestBlock.OnPlace` returns
   `PlaceResult{OK: true, Writes: ..., BEAdds: []world.BlockEntity{&Chest{...}}}`.
