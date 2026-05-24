package parsers

import (
	tc "github.com/Gagonlaire/mcgoserv/internal/mc/textcomponent"
	"github.com/Gagonlaire/mcgoserv/internal/mcdata"
	"github.com/Gagonlaire/mcgoserv/internal/proto"
	"github.com/Gagonlaire/mcgoserv/internal/systems/commander"
)

type Registry interface {
	WireName() proto.Identifier
	Lookup(path proto.Identifier) (any, bool)
}

// todo: add missing registries:
//   - mob_effect      (minecraft:mob_effect)     — /effect give|clear
//   - enchantment     (minecraft:enchantment)    — /enchant reloadable
//   - fluid           (minecraft:fluid)
//   - particle_type   (minecraft:particle_type)  — /particle
//   - game_event      (minecraft:game_event)
//   - potion          (minecraft:potion)         — reloadable
//   - banner_pattern  (minecraft:banner_pattern) — reloadable
//   - biome           (minecraft:worldgen/biome) — reloadable
//   - dimension       (minecraft:dimension)      — reloadable
//   - loot_table      (minecraft:loot_table)     — reloadable
//   - recipe          (minecraft:recipe)         — reloadable
//   - advancement     (minecraft:advancement)    — reloadable
//   - predicate       (minecraft:predicate)      — reloadable
//   - damage_type     (minecraft:damage_type)    — reloadable

func readIdentifier(r *commander.CommandReader) (proto.Identifier, int, error) {
	start := r.Cursor()
	raw := r.ReadWord()
	if raw == "" {
		return "", start, commander.NewParsingErrorAt(r, tc.Translatable(mcdata.ArgumentIdInvalid, tc.Text(raw)), start)
	}
	id, err := proto.ParseIdentifier(raw)
	if err != nil {
		r.SetCursor(start)
		return "", start, commander.NewParsingErrorAt(r, tc.Translatable(mcdata.ArgumentIdInvalid, tc.Text(raw)), start)
	}
	return id, start, nil
}
