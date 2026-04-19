package world

import (
	"math"
	"math/rand/v2"
	"strings"

	"github.com/Gagonlaire/mcgoserv/internal/mc"
	"github.com/Gagonlaire/mcgoserv/internal/mc/entities"
	"github.com/google/uuid"
)

// ResolveTarget resolves a target to all matching entities. Player-only selectors
// (@p, @r, @a) -> player index, entity selectors (@e, @n) -> full entity index. @s -> source
// UUID -> full entity index, name targets only match players
func (w *World) ResolveTarget(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []entities.Entity {
	switch target.Type {
	case mc.TargetTypePlayerName:
		for _, p := range w.PlayersByID {
			if p.Name == target.Name {
				return []entities.Entity{p}
			}
		}
		return nil
	case mc.TargetTypeUUID:
		if e := w.EntitiesByUUID[target.UUID]; e != nil {
			return []entities.Entity{e}
		}
		return nil
	case mc.TargetTypeSelector:
		return w.resolveSelector(target.Selector, sourceUUID, sourcePos)
	}
	return nil
}

func (w *World) ResolvePlayers(target *mc.EntityTarget, sourceUUID uuid.UUID, sourcePos [3]float64) []*entities.Player {
	ents := w.ResolveTarget(target, sourceUUID, sourcePos)
	if len(ents) == 0 {
		return nil
	}
	players := make([]*entities.Player, 0, len(ents))
	for _, e := range ents {
		if p, ok := e.(*entities.Player); ok {
			players = append(players, p)
		}
	}
	return players
}

func (w *World) resolveSelector(sel *mc.Selector, sourceUUID uuid.UUID, sourcePos [3]float64) []entities.Entity {
	switch sel.Variable {
	case mc.SelectorVariableSelf:
		if e := w.EntitiesByUUID[sourceUUID]; e != nil {
			return []entities.Entity{e}
		}
		return nil
	case mc.SelectorVariableAllPlayers:
		out := make([]entities.Entity, 0, len(w.PlayersByID))
		for _, p := range w.PlayersByID {
			out = append(out, p)
		}
		return out
	case mc.SelectorVariableAllEntities:
		out := make([]entities.Entity, 0, len(w.EntitiesByID))
		for _, e := range w.EntitiesByID {
			out = append(out, e)
		}
		return out
	case mc.SelectorVariableNearestPlayer:
		if p := w.nearestPlayer(sourcePos); p != nil {
			return []entities.Entity{p}
		}
		return nil
	case mc.SelectorVariableNearestEntity:
		if e := w.nearestEntity(sourcePos); e != nil {
			return []entities.Entity{e}
		}
		return nil
	case mc.SelectorVariableRandomPlayer:
		players := w.Players()
		if len(players) == 0 {
			return nil
		}
		return []entities.Entity{players[rand.IntN(len(players))]}
	}
	return nil
}

func (w *World) nearestEntity(pos [3]float64) entities.Entity {
	var nearest entities.Entity
	bestDist := math.MaxFloat64
	for _, e := range w.EntitiesByID {
		d := distSq(pos, e.Base().Position)
		if d < bestDist {
			bestDist = d
			nearest = e
		}
	}
	return nearest
}

func (w *World) nearestPlayer(pos [3]float64) *entities.Player {
	var nearest *entities.Player
	bestDist := math.MaxFloat64
	for _, p := range w.PlayersByID {
		d := distSq(pos, p.Position)
		if d < bestDist {
			bestDist = d
			nearest = p
		}
	}
	return nearest
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
			p, ok := e.(*entities.Player)
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
