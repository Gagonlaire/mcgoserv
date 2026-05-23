# Commands

Conventions for the `commands` package. Every command registration lives here

- https://minecraft.wiki/w/Commands
- https://minecraft.wiki/w/Java_Edition_protocol/Command_data

## File layout

- **One file per top-level command.** `kill.go`, `summon.go`, `time.go`, …
- **Family grouping** is allowed when commands are similar:
  - `chat.go` — `msg`/`tell`/`w`, `say`, `teammsg`/`tm`, `tellraw`
  - `ban.go` — `ban`/`ban-ip`/`pardon`/`pardon-ip`
  - `op.go` — `op`/`deop`
  - `whitelist.go` — all `whitelist` subcommands
- **`common.go` holds reusable helpers, no command specific logic.** If a helper is only used by one command,
  put it in that file instead.
- **`bootstrap.go`** wires every `register*` into `RegisterAll`. Keep the call order alphabetical.

## Registration shape

```go
func registerKill(s *server.Server) {
    s.Commander.RegisterBuilders(func() {
        Build("/kill <target>", parsers.Entity).
            Requires(2).Executes(killExecutor(s))
    })
}
```

The Builder chain is read left-to-right in this order:

```
Build(syntax, parsers...) → Aliases(...) → Requires(N) → Description(...) → Executes(fn) | ExecutesEach(fn...)
```

### Aliases

Use `.Aliases("...")` inside the `RegisterBuilders` block. Never call
`Register(Literal(alias).Redirects(...))` afterwards.

```go
Build("/msg <targets> <message ...>", parsers.Entity.PlayersOnly(true), parsers.Message).
    Aliases("tell", "w").Executes(sendMsg(s))
```

Aliases target the **root** of the command

### Executor style

**Factory by default.** Long bodies live in a named function so `register*` stays
purely structural:

```go
func registerGamemode(s *server.Server) {
    s.Commander.RegisterBuilders(func() {
        Build("/gamemode <mode> [<target>]",
            parsers.Gamemode, parsers.Entity.PlayersOnly(true),
        ).Requires(2).Executes(setGamemode(s))
    })
}

func setGamemode(s *server.Server) Command {
    return func(cc *CommandContext) (*CommandResult, error) { … }
}
```

**Inline closures only for simple bodies** (~5-20 LOC).

## When to drop to the imperative API

`Build()` covers the common cases — literals, required/optional args, choices, and
linear paths that share prefixes (prefix merging happens automatically). Drop to
the regular api when the command needs:

- **Complex branching** that can't be expressed as a shared prefix with `Build()`.
- **Redirects** other than root-aliases (root-aliases are handled by `.Aliases(...)`)
- **Forks** or custom `RedirectModifier`s
- **Custom suggestion providers** beyond what parsers expose
- **Shared state on intermediate nodes** — when multiple branches need to read or
  attach behavior on the same internal node, not just pass through it

## Source helpers

Always use the canonical helpers in `common.go`:

| Helper | Returns | Use for |
|--------|---------|---------|
| `commandSource(cc)` | `(uuid.UUID, [3]float64)` | Passing source identity to world resolvers |
| `actorName(cc)` | `string` (player name or `"Server"`) | Audit-log fields (ban source, kick reason author) |
| `entityDisplayName(e)` | `tc.Component` | Rendering an entity in feedback messages |
| `resolveProfileTargets(s, cc, target)` | `[]ProfileTarget` | Flattening a GameProfile/EntitySelector parse into `{UUID, Name}` pairs; handles online-mode lookup and offline UUID fallback |

Do not reinvent these inline. Prefer extending helpers instead of duplicating code.
