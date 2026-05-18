package world

import (
	"math"
	"math/rand/v2"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entity"
	"github.com/google/uuid"
)

// ResolveTarget resolves a target to all matching entities. Player-only selectors
// (@p, @r, @a) -> player index, entity selectors (@e, @n) -> full entity index. @s -> source
// UUID -> full entity index, name targets only match players
func (w *World) ResolveTarget(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []entity.Entity {
	switch target.Type {
	case mc.TargetTypePlayerName:
		for _, p := range w.PlayersByID {
			if p.Name == target.Name {
				return []entity.Entity{p}
			}
		}
		return nil
	case mc.TargetTypeUUID:
		if e := w.EntitiesByUUID[target.UUID]; e != nil {
			return []entity.Entity{e}
		}
		return nil
	case mc.TargetTypeSelector:
		return w.resolveSelector(target.Selector, sourceUUID, sourcePos)
	}
	return nil
}

func (w *World) ResolvePlayers(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []*entity.Player {
	ents := w.ResolveTarget(target, sourceUUID, sourcePos)
	if len(ents) == 0 {
		return nil
	}
	players := make([]*entity.Player, 0, len(ents))
	for _, e := range ents {
		if p, ok := e.(*entity.Player); ok {
			players = append(players, p)
		}
	}
	return players
}

func (w *World) resolveSelector(sel *mc.Selector, sourceUUID uuid.UUID, sourcePos [3]float64) []entity.Entity {
	if sel.Variable == mc.SelectorVariableSelf {
		e := w.EntitiesByUUID[sourceUUID]
		if e == nil || !passesFilter(e, sel) {
			return nil
		}
		return []entity.Entity{e}
	}

	candidates := w.selectorCandidates(sel)
	filtered := candidates[:0]
	for _, e := range candidates {
		if passesFilter(e, sel) {
			filtered = append(filtered, e)
		}
	}
	return reduceSelectorResult(sel, filtered, sourcePos)
}

func (w *World) selectorCandidates(sel *mc.Selector) []entity.Entity {
	switch sel.Variable {
	case mc.SelectorVariableAllPlayers,
		mc.SelectorVariableNearestPlayer,
		mc.SelectorVariableRandomPlayer:
		out := make([]entity.Entity, 0, len(w.PlayersByID))
		for _, p := range w.PlayersByID {
			out = append(out, p)
		}
		return out
	case mc.SelectorVariableAllEntities,
		mc.SelectorVariableNearestEntity:
		out := make([]entity.Entity, 0, len(w.EntitiesByID))
		for _, e := range w.EntitiesByID {
			out = append(out, e)
		}
		return out
	}
	return nil
}

func reduceSelectorResult(sel *mc.Selector, filtered []entity.Entity, sourcePos [3]float64) []entity.Entity {
	switch sel.Variable {
	case mc.SelectorVariableAllPlayers, mc.SelectorVariableAllEntities:
		return filtered
	case mc.SelectorVariableNearestPlayer, mc.SelectorVariableNearestEntity:
		return nearestOf(filtered, sourcePos)
	case mc.SelectorVariableRandomPlayer:
		if len(filtered) == 0 {
			return nil
		}
		return []entity.Entity{filtered[rand.IntN(len(filtered))]}
	}
	return nil
}

func nearestOf(ents []entity.Entity, pos [3]float64) []entity.Entity {
	var nearest entity.Entity
	bestDist := math.MaxFloat64
	for _, e := range ents {
		d := distSq(pos, e.Base().Position)
		if d < bestDist {
			bestDist = d
			nearest = e
		}
	}
	if nearest == nil {
		return nil
	}
	return []entity.Entity{nearest}
}

// passesFilter reports whether entity e satisfies every filter argument in sel.
// Filter args are applied independently of the selector variable; the variable
// only governs candidate enumeration and reduction.
func passesFilter(e entity.Entity, sel *mc.Selector) bool {
	if !passesTypeFilter(e, sel) {
		return false
	}
	return true
}

func passesTypeFilter(e entity.Entity, sel *mc.Selector) bool {
	if !sel.TypeInclude.Present && len(sel.TypeExclude) == 0 {
		return true
	}
	name := e.GetType().Name()
	if sel.TypeInclude.Present && sel.TypeInclude.Value != name {
		return false
	}
	for _, ex := range sel.TypeExclude {
		if ex == name {
			return false
		}
	}
	return true
}

func (w *World) ResolveMessage(msg *mc.ParsedMessage, sourceUUID uuid.UUID, sourcePos [3]float64) string {
	if len(msg.Selectors) == 0 {
		return msg.Raw
	}
	var b strings.Builder
	b.Grow(len(msg.Raw))
	prev := 0
	for _, span := range msg.Selectors {
		b.WriteString(msg.Raw[prev:span.Start])
		ents := w.resolveSelector(span.Selector, sourceUUID, sourcePos)
		first := true
		for _, e := range ents {
			p, ok := e.(*entity.Player)
			if !ok {
				continue
			}
			if !first {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
			first = false
		}
		prev = span.End
	}
	b.WriteString(msg.Raw[prev:])
	return b.String()
}

func distSq(a, b [3]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	dz := a[2] - b[2]
	return dx*dx + dy*dy + dz*dz
}
